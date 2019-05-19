package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/mail"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/jhillyerd/enmime"
	"golang.org/x/xerrors"
)

// insert saves an image to the database and the filesystem
func insert(in io.Reader) (err error) {
	// Open file to save mail
	f, err := newMailFile()
	if err != nil {
		return xerrors.Errorf("can not open mail file for writing: %w", err)
	}
	defer f.Close()

	// Save mail to file as soon as `in` is read
	in = io.TeeReader(in, f)

	// If an error happens after this line, move mail to error folder
	defer func() {
		if err != nil {
			if e := f.move("error"); e != nil {
				log.Printf("Original error: %v", err)
				err = xerrors.Errorf("can not move mail to folder error: %w", e)
			}
		}
	}()

	// Build enmime envelope by reading `in`
	envelope, err := enmime.ReadEnvelope(in)
	if err != nil {
		return xerrors.Errorf("can not interprete mail: %w", err)
	}

	// Read the senders address
	from, err := mail.ParseAddress(envelope.Root.Header.Get("from"))
	if err != nil {
		return xerrors.Errorf("can not interprete mail address: %w", err)
	}

	// If an error happens after this line, send a respond mail
	defer func() {
		if err != nil {
			// TODO: Don't send the error template but a "500" template
			sendErr := respondError(from.Name, from.Address, "Fehler", []error{errInternal})
			if sendErr != nil {
				log.Printf("Original error: %v", err)
				err = xerrors.Errorf("can not send an error mail: %w", sendErr)
			}
		}
	}()

	// Parse the mail and get the relevant informations
	// TODO: Try to return image as reader or writer
	subject, text, imageExt, image, thumbnail, errs := parseMail(envelope)
	if len(errs) > 0 {
		if err := f.move("invalid"); err != nil {
			return xerrors.Errorf("can not move mail to invalid folder: %w", err)
		}

		if err := respondError(from.Name, from.Address, subject, errs); err != nil {
			return xerrors.Errorf("can not responde to invalid mail: %w", err)
		}
		return nil
	}

	pool, err := newPool(redisAddr)
	if err != nil {
		return xerrors.Errorf("can not create redis pool: %w", err)
	}

	// Save data to redis
	id, token, err := pool.postEntry(from.Name, from.Address, subject, text, imageExt, thumbnail)
	if err != nil {
		return xerrors.Errorf("can not save mail: %w", err)
	}

	// If an error happens after this line, delete element from redis
	defer func() {
		if err != nil {
			// TODO: Delete element from redis
		}
	}()

	// Save image to disk
	imagePath := path.Join(mailimagePath(), "images", fmt.Sprintf("%d%s", id, imageExt))
	err = ioutil.WriteFile(imagePath, image, 0644)
	if err != nil {
		return xerrors.Errorf("can not save image to disk: %w", err)
	}

	// If an error happens after this line, delete image from disk
	defer func() {
		if err != nil {
			// TODO: Delete element from disk
		}
	}()

	// Move mail to success
	if err := f.move("success"); err != nil {
		return xerrors.Errorf("can not move mail to success folder: %w", err)
	}

	if err := respondSuccess(from.Name, from.Address, subject, token); err != nil {
		return xerrors.Errorf("can not send success mail: %w", err)
	}
	return nil
}

// parseMail parses an email and returns all relevant information about it
func parseMail(mail *enmime.Envelope) (subject, text, imageExt string, image, thumbnail []byte, errs []error) {
	errs = make([]error, 0)

	subject = strings.TrimSpace(mail.GetHeader("subject"))
	subject = strings.TrimPrefix(subject, "***SPAM***")
	if len(subject) > subjectLength {
		errs = append(errs, errLongSubject)
	}

	text = regexp.MustCompile(`\r?\n`).ReplaceAllString(mail.Text, " ")
	text = strings.TrimSpace(text)
	if len(text) > textLength {
		errs = append(errs, errLongText)
	}

	image, thumbnail, imageExt, err := parseAttachments(mail.Root)
	if err != nil {
		errs = append(errs, err)
	}

	return subject, text, imageExt, image, thumbnail, errs
}

// parseAttachments parses all attachments from an mail body and looks for supported images
// The returned error message is send to the user.
// Returns the image and the thumbnail as byte and the file extension
func parseAttachments(part *enmime.Part) ([]byte, []byte, string, error) {
	imageParts := part.DepthMatchAll(imageMatcher)
	if len(imageParts) < 1 {
		return nil, nil, "", errNoImage
	}

	if len(imageParts) > 1 {
		return nil, nil, "", errMultiImage
	}

	if len(imageParts[0].Errors) > 0 {
		errs := make([]string, len(imageParts[0].Errors))
		for i, err := range imageParts[0].Errors {
			errs[i] = err.Error()
		}
		log.Printf("Can not parse image: %s", strings.Join(errs, ","))
		return nil, nil, "", errParsingImage
	}

	imageObject, err := imaging.Decode(bytes.NewReader(imageParts[0].Content))
	if err != nil {
		log.Printf("Can not read image: %v", err)
		return nil, nil, "", errParsingImage
	}

	imageObject = imaging.Fill(imageObject, 250, 200, imaging.Center, imaging.Lanczos)
	buf := bytes.NewBuffer(make([]byte, 0))
	if err := imaging.Encode(buf, imageObject, imaging.JPEG); err != nil {
		log.Printf("Can not create thumbnail: %v", err)
		return nil, nil, "", errCreateThumbnail
	}
	return imageParts[0].Content, buf.Bytes(), filepath.Ext(imageParts[0].FileName), nil
}

// isAllowed returns true, if the attachment has a supported mail format
func isAllowed(contentType string) bool {
	for _, format := range allowedFormats {
		if ("image/" + format) == contentType {
			return true
		}
	}
	return false
}

// imageMatcher is a helper for the enime api to find image attachments in a mail
func imageMatcher(part *enmime.Part) bool {
	return isAllowed(part.ContentType)
}
