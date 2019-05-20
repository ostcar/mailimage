package mailimage

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/mail"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jhillyerd/enmime"
	"golang.org/x/xerrors"
)

// Insert saves an image to the database and the filesystem
func Insert(in io.Reader) error {
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
				log.Printf("Can not move mail to error folder: %v", e)
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
			if e := respondError(from.Name, from.Address, "Fehler", []error{errInternal}); e != nil {
				log.Printf("Can not send an error mail: %v", e)
			}
		}
	}()

	// Parse the mail and get the relevant informations
	subject, text, imageExt, image, errs := parseMail(envelope)
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
		return xerrors.Errorf("can not create redis pool to same mail %s: %w", f.name, err)
	}

	// Save data to redis
	id, token, err := pool.postEntry(from.Name, from.Address, subject, text, imageExt)
	if err != nil {
		return xerrors.Errorf("can not save mail: %w", err)
	}

	// If an error happens after this line, delete element from redis
	defer func() {
		if err != nil {
			if e := pool.deleteFromID(id); e != nil {
				log.Printf("Cat not remove element %d from redis: %v", id, err)
			}
		}
	}()

	// Save image to disk
	if err := os.MkdirAll(path.Join(mailimagePath(), "images"), os.ModePerm); err != nil {
		return xerrors.Errorf("can not create folder images: %w", err)
	}

	imagePath := path.Join(mailimagePath(), "images", fmt.Sprintf("%d%s", id, imageExt))
	err = ioutil.WriteFile(imagePath, image, 0644)
	if err != nil {
		return xerrors.Errorf("can not save image to disk: %w", err)
	}

	// If an error happens after this line, delete image from disk
	defer func() {
		if err != nil {
			if e := os.Remove(imagePath); e != nil {
				log.Printf("Can not remove image: %v", e)
			}
		}
	}()

	// Move mail to success and rename it to its id
	if err := f.rename(strconv.Itoa(id)); err != nil {
		return err
	}

	if err := f.move("success"); err != nil {
		return err
	}

	if err := respondSuccess(from.Name, from.Address, subject, token); err != nil {
		return xerrors.Errorf("can not send success mail: %w", err)
	}
	return nil
}

// parseMail parses an email and returns all relevant information about it
func parseMail(mail *enmime.Envelope) (subject, text, imageExt string, image []byte, errs []error) {
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

	image, imageExt, err := parseAttachments(mail.Root)
	if err != nil {
		errs = append(errs, err)
	}

	return subject, text, imageExt, image, errs
}

// parseAttachments parses all attachments from an mail body and looks for supported images
// The returned error message is send to the user.
// Returns the image and the file extension
func parseAttachments(part *enmime.Part) ([]byte, string, error) {
	imageParts := part.DepthMatchAll(imageMatcher)
	if len(imageParts) < 1 {
		return nil, "", errNoImage
	}

	if len(imageParts) > 1 {
		return nil, "", errMultiImage
	}

	if len(imageParts[0].Errors) > 0 {
		errs := make([]string, len(imageParts[0].Errors))
		for i, err := range imageParts[0].Errors {
			errs[i] = err.Error()
		}
		log.Printf("Can not parse image: %s", strings.Join(errs, ","))
		return nil, "", errParsingImage
	}
	return imageParts[0].Content, filepath.Ext(imageParts[0].FileName), nil
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
