package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"golang.org/x/xerrors"
)

type pool struct {
	redis.Pool
}

// newPool creates a new redis pool
func newPool(addr string) (*pool, error) {
	p := pool{redis.Pool{
		MaxActive:   100,
		Wait:        true,
		MaxIdle:     10,
		IdleTimeout: 240 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", addr) },
	}}

	// Test connection
	conn := p.Get()
	defer conn.Close()

	_, err := conn.Do("PING")
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// postEntry saves an new entry to the database
// Returns the new id and the delete token
func (p *pool) postEntry(fromName, fromAddr, subject, text, imageExt string, thumbnail []byte) (int, string, error) {
	conn := p.Get()
	defer conn.Close()

	id, err := p.getNewID()
	if err != nil {
		return 0, "", err
	}

	_, err = conn.Do(
		"HMSET",
		key("entry", strconv.Itoa(id)),
		"from",
		fromName,
		"mail",
		fromAddr,
		"subject",
		subject,
		"text",
		text,
		"fileext",
		imageExt,
		"thumbnail",
		thumbnail,
		"created",
		time.Now().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return 0, "", xerrors.Errorf("can not post entry: %w", err)
	}

	if _, err = conn.Do("SADD", key("entries"), id); err != nil {
		return 0, "", xerrors.Errorf("can not save entry id: %s", err)
	}

	token, err := p.createDeleteToken(id)
	if err != nil {
		return 0, "", err
	}
	return id, token, nil
}

// listEntries gets all enties from the database
func (p *pool) listEntries() ([]entry, error) {
	conn := p.Get()
	defer conn.Close()

	ids, err := redis.Ints(conn.Do("SMEMBERS", key("entries")))
	if err != nil {
		return nil, xerrors.Errorf("can not receive ids: %w", err)
	}

	var entries []entry
	for _, id := range ids {
		values, err := redis.Strings(conn.Do(
			"HMGET",
			key("entry", strconv.Itoa(id)),
			"from",
			"subject",
			"text",
			"fileext",
			"created",
		))
		if err != nil {
			return nil, xerrors.Errorf("can not receice entry %d: %w", id, err)
		}

		created, err := time.Parse("2006-01-02 15:04:05", values[4])
		if err != nil {
			return nil, xerrors.Errorf("can not parse created time: %w", err)
		}

		entries = append(entries, entry{
			ID:        id,
			From:      values[0],
			Subject:   values[1],
			Text:      values[2],
			Extension: values[3],
			Created:   created.Format("2006-01-02 15:04"),
		})
	}
	return entries, nil
}

// getImage gets an image and the file extension for an id
// TODO: Return an reader, Only return the extension, the image is not saved in redis
func (p *pool) getImage(id int) ([]byte, string, error) {
	conn := p.Get()
	defer conn.Close()

	ext, err := redis.String(conn.Do("HGET", key("entry", strconv.Itoa(id)), "fileext"))
	if err != nil {
		return nil, "", xerrors.Errorf("can not get extension for image with id %d: %w", id, err)
	}

	// The image is unknown in redis
	if ext == "" {
		return nil, "", errUnknownImage
	}

	filePath := path.Join(mailimagePath(), "images", fmt.Sprintf("%d%s", id, ext))
	image, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", errUnknownImage
		}
		return nil, "", xerrors.Errorf("can not get image %d: %s", id, err)
	}

	return image, ext, nil
}

// getThumbnail gets an thumbnail for an id
// TODO: Save thumbnail on disk. For this purpose it is not necessary to hold the thumbnails in memory all the time
func (p *pool) getThumbnail(id int) ([]byte, error) {
	conn := p.Get()
	defer conn.Close()

	image, err := redis.Bytes(conn.Do("HGET", key("entry", strconv.Itoa(id)), "thumbnail"))
	if err != nil {
		return nil, xerrors.Errorf("can not get thumbnail %d: %w", id, err)
	}

	if image == nil {
		return nil, errUnknownImage
	}
	return image, nil
}

// createDeleteToken saves a new delete token into the database
func (p *pool) createDeleteToken(id int) (string, error) {
	conn := p.Get()
	defer conn.Close()

	token := genToken()
	// TODO: Test that token does not exist

	_, err := conn.Do("SET", key("deletetoken", token), id, "EX", tokenExpire)
	if err != nil {
		return "", xerrors.Errorf("can not generate delete token: %w", err)
	}
	return token, nil
}

// deleteFromToken deletes a entry from a delete token
func (p *pool) deleteFromToken(token string) error {
	conn := p.Get()
	defer conn.Close()

	id, err := redis.Int(conn.Do("GET", key("deletetoken", token)))
	if err != nil {
		return xerrors.Errorf("can not find id for token: %w", err)
	}

	if id == 0 {
		return errUnknownImage
	}

	return p.deleteFromID(id)
}

// deleteFromID deletes an entry from an id
func (p *pool) deleteFromID(id int) error {
	conn := p.Get()
	defer conn.Close()

	ext, err := redis.String(conn.Do("HGET", key("entry", strconv.Itoa(id)), "fileext"))
	if err != nil {
		return xerrors.Errorf("can not get image extension %d: %w", id, err)
	}

	// Delete from list
	if _, err := conn.Do("SREM", key("entries"), id); err != nil {
		return xerrors.Errorf("can not delete entry id: %w", err)
	}

	// Delete from redis
	if _, err := conn.Do("DEL", key("entry", strconv.Itoa(id))); err != nil {
		return xerrors.Errorf("can not delete entry: %w", err)
	}

	// Delete image from disk
	filePath := path.Join(mailimagePath(), "images", fmt.Sprintf("%d%s", id, ext))
	if err := os.Remove(filePath); err != nil {
		return xerrors.Errorf("can not delete image from disk: %w", err)
	}

	// Delete mail fom disk
	filePath = path.Join(mailimagePath(), "success", strconv.Itoa(id))
	if err := os.Remove(filePath); err != nil {
		return xerrors.Errorf("can not delete file from disk: %w", err)
	}
	return nil
}

// getNewID creates a new database id
func (p *pool) getNewID() (int, error) {
	conn := p.Get()
	defer conn.Close()

	id, err := redis.Int(conn.Do("INCR", key("last_id")))
	if err != nil {
		return 0, xerrors.Errorf("can not get new id: %w", err)
	}
	return id, nil
}

// key creates a redis key with a prefix from a list of strings
func key(keys ...string) string {
	return fmt.Sprintf("%s:%s", "mailimage", strings.Join(keys, ":"))
}

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func genToken() string {
	b := make([]byte, tokenLength)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
