package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/mail"
	"os"
	"regexp"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/jhillyerd/enmime"
)

var AllowedFormats = [...]string{
	"jpeg",
	"png",
	"jpg",
}

func isAllowed(contentType string) bool {
	for _, format := range AllowedFormats {
		if ("image/" + format) == contentType {
			return true
		}
	}
	return false
}

func imageMatcher(part *enmime.Part) bool {
	return isAllowed(part.ContentType)
}

func parseAttachments(part *enmime.Part) (image, thumbnail []byte, filename string, err error) {
	imageParts := part.DepthMatchAll(imageMatcher)
	if len(imageParts) < 1 {
		err = fmt.Errorf("Keine Bilddatei in der E-Mail gefunden.")
		return
	}
	if len(imageParts) > 1 {
		err = fmt.Errorf("Mehrere Bilder gefunden. Die E-Mail darf maximal ein Bild entalten.")
		return
	}

	image, err = ioutil.ReadAll(imageParts[0])
	if err != nil {
		err = fmt.Errorf("Die Bilddatei kann nicht gelesen werden: %s", err)
		return
	}

	filename = imageParts[0].FileName

	imageObject, err := imaging.Decode(bytes.NewReader(image))
	if err != nil {
		err = fmt.Errorf("Die Bilddatei kann nicht gelesen werden: %s", err)
		return
	}

	thumbnailObject := imaging.Fill(imageObject, 250, 200, imaging.Center, imaging.Lanczos)
	buf := bytes.NewBuffer(make([]byte, 0))
	err = imaging.Encode(buf, thumbnailObject, imaging.JPEG)
	if err != nil {
		err = fmt.Errorf("Thumbnail kann aus dem Bild nicht erstellt werden: %s", err)
		return
	}
	thumbnail = buf.Bytes()
	return
}

func parseMail(mail *enmime.Envelope) (subject, text, imageName string, image, thumbnail []byte, messages []string) {
	messages = make([]string, 0)

	subject = strings.TrimSpace(mail.Root.Header.Get("subject"))
	subject = strings.TrimPrefix(subject, "***SPAM***")
	if len([]rune(subject)) > subjectLength {
		messages = append(messages, fmt.Sprintf("E-Mail Betreff ist zu lang. Maximal %d zeichen sind erlaubt.", subjectLength))
	}

	text = regexp.MustCompile(`\r?\n`).ReplaceAllString(mail.Text, " ")
	text = strings.TrimSpace(text)
	if len([]rune(text)) > textLength {
		messages = append(messages, fmt.Sprintf("Text der E-Mail darf maximal %d Zeichen lang sein.", textLength))
	}

	image, thumbnail, imageName, err := parseAttachments(mail.Root)
	if err != nil {
		messages = append(messages, err.Error())
	}

	return
}

func insert(in io.Reader) {
	var raw []byte
	var from *mail.Address
	var err error

	defer func() {
		if r := recover(); r != nil {
			recoveredErr := r.(string)
			err := saveRawMail(raw, recoveredErr)
			if err != nil {
				log.Fatalf("Can not read mail: %s", r)
			}
			err = sendError(from, "Fehler", []string{recoveredErr})
			if err != nil {
				log.Fatalf("Can not send mail response: %s", r)
			}
		}
	}()

	// Log to file
	if !debug {
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			log.Fatalf("Can not open logfile: %s", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	// Read mail to raw:
	raw, err = ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Can not read mail from stdin: %s", err)
	}

	// Build enmime envelope from raw mail
	envelope, err := enmime.ReadEnvelope(bytes.NewReader(raw))
	if err != nil {
		log.Panicf("Can not interprete mail: %s", err)
	}

	// Read the senders address
	from, err = mail.ParseAddress(envelope.Root.Header.Get("from"))
	if err != nil {
		log.Panicf("Can not interprete mail address: %s", err)
	}

	// Parse the mail and get all relevant informations
	subject, text, imageName, image, thumbnail, messages := parseMail(envelope)
	if len(messages) > 0 {
		err = sendError(from, subject, messages)
		if err != nil {
			log.Panicf("Can not send response mail: %s", err)
		}
		return
	}

	entry := &Entry{raw, from, subject, text, image, imageName, thumbnail}
	_, token, err := postEntry(entry)
	if err != nil {
		log.Panicf("Can not save mail: %s", err)
	}

	err = sendSuccess(from, subject, token)
	if err != nil {
		log.Panicf("can not send success mail: %s", err)
	}
}
