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
)

var redisPool *redis.Pool

func init() {
	CreateRedisPool(":6379")
	rand.Seed(time.Now().UTC().UnixNano())
}

// CreateRedisPool sets the redis pool to connect to the host.
func CreateRedisPool(host string) {
	redisPool = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		MaxActive:   20,
		Wait:        true,
		Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", host) },
	}
}

func key(keys ...string) string {
	return fmt.Sprintf("%s:%s", "mailimage", strings.Join(keys, ":"))
}

func getNewID() (int, error) {
	conn := redisPool.Get()
	defer conn.Close()

	id, err := redis.Int(conn.Do("INCR", key("last_id")))
	if err != nil {
		return 0, fmt.Errorf("can not get new id: %s", err)
	}
	return id, nil
}

func postEntry(entry *Entry) (int, string, error) {
	conn := redisPool.Get()
	defer conn.Close()

	id, err := getNewID()
	if err != nil {
		return 0, "", err
	}

	_, err = conn.Do(
		"HMSET",
		key("entry", strconv.Itoa(id)),
		"from",
		entry.address.Name,
		"mail",
		entry.address.Address,
		"subject",
		entry.subject,
		"text",
		entry.text,
		"fileext",
		entry.imageExt,
		"thumbnail",
		entry.thumbnail,
		"created",
		time.Now().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return 0, "", fmt.Errorf("can not post entry: %s", err)
	}
	_, err = conn.Do("SADD", key("entries"), id)
	if err != nil {
		return 0, "", fmt.Errorf("can not save entry id: %s", err)
	}

	token, err := createDeleteToken(id)
	if err != nil {
		return 0, "", err
	}
	return id, token, nil
}

func listEntries() ([]ServeEntry, error) {
	conn := redisPool.Get()
	defer conn.Close()

	ids, err := redis.Ints(conn.Do("SMEMBERS", key("entries")))
	if err != nil {
		return nil, fmt.Errorf("can not receive ids: %s", err)
	}
	entries := make([]ServeEntry, 0)
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
			return nil, fmt.Errorf("can not receice entry %d: %s", id, err)
		}

		created, err := time.Parse("2006-01-02 15:04:05", values[4])
		if err != nil {
			return nil, fmt.Errorf("can not parse created time")
		}
		entries = append(entries, ServeEntry{
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

func getImage(id int) ([]byte, string, error) {
	conn := redisPool.Get()
	defer conn.Close()

	ext, err := redis.String(conn.Do("HGET", key("entry", strconv.Itoa(id)), "fileext"))
	if err != nil {
		return nil, "", fmt.Errorf("can not get imageext %d: %s", id, err)
	}

	filePath := path.Join(mailimagePath, "images", fmt.Sprintf("%d%s", id, ext))
	image, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("can not get image %d: %s", id, err)
	}

	return image, ext, nil
}

func getThumbnail(id int) ([]byte, error) {
	conn := redisPool.Get()
	defer conn.Close()

	image, err := redis.Bytes(conn.Do("HGET", key("entry", strconv.Itoa(id)), "thumbnail"))
	if err != nil {
		return nil, fmt.Errorf("can not get image %d: %s", id, err)
	}
	return image, nil
}

func getNewToken() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, tokenLength)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	// TODO: test that token is not in use
	return string(b)
}

func createDeleteToken(id int) (string, error) {
	conn := redisPool.Get()
	defer conn.Close()

	token := getNewToken()

	_, err := conn.Do("SET", key("deletetoken", token), id, "EX", tokenExpire)
	if err != nil {
		return "", fmt.Errorf("can not generate delete token: %s", err)
	}
	return token, nil
}

func deleteFromToken(token string) error {
	conn := redisPool.Get()
	defer conn.Close()

	id, err := redis.Int(conn.Do("GET", key("deletetoken", token)))
	if err != nil {
		return fmt.Errorf("can not find id for token: %s", err)
	}

	return deleteFromID(id)
}

func deleteFromID(id int) error {
	conn := redisPool.Get()
	defer conn.Close()

	ext, err := redis.String(conn.Do("HGET", key("entry", strconv.Itoa(id)), "fileext"))
	if err != nil {
		return fmt.Errorf("can not get imageext %d: %s", id, err)
	}

	// Delete from list
	_, err = conn.Do("SREM", key("entries"), id)
	if err != nil {
		return fmt.Errorf("can not delete entry id: %s", err)
	}

	// Delete from redis
	_, err = conn.Do("DEL", key("entry", strconv.Itoa(id)))
	if err != nil {
		return fmt.Errorf("can not delete entry: %s", err)
	}

	// Delete image from disk
	filePath := path.Join(mailimagePath, "images", fmt.Sprintf("%d%s", id, ext))
	err = os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("can not delete image from disk: %s", err)
	}

	// Delete mail fom disk
	filePath = path.Join(mailimagePath, "success", strconv.Itoa(id))
	err = os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("can not delete file from disk: %s", err)
	}
	return nil
}
