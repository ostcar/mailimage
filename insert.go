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

func parseAuthor(author string) (*mail.Address, []error) {
	errs := make([]error, 0)
	address, err := mail.ParseAddress(author)
	if err != nil {
		errs = append(errs, fmt.Errorf("Von ungültiger E-Mail-Adresse gesendet."))
	}

	if len(errs) > 0 {
		return nil, errs
	}
	return address, nil
}

func parseSubject(subject string) (string, []error) {
	errs := make([]error, 0)
	if len([]rune(subject)) > 25 {
		errs = append(errs, fmt.Errorf("E-Mail Betreff ist zu lang. Maximal 25 zeichen sind erlaubt."))
	}
	if len(errs) > 0 {
		return "", errs
	}
	subject = strings.TrimSpace(subject)
	return subject, nil
}

func parseText(text string) (string, []error) {
	errs := make([]error, 0)
	if len([]rune(text)) > 200 {
		errs = append(errs, fmt.Errorf("Text der E-Mail darf maximal 200 Zeichen lang sein."))
	}
	text = regexp.MustCompile(`\r?\n`).ReplaceAllString(text, " ")

	if len(errs) > 0 {
		return "", errs
	}
	text = strings.TrimSpace(text)
	return text, nil
}

func parseAttachments(part *enmime.Part) (image, thumbnail []byte, filename string, errors []error) {
	errs := make([]error, 0)
	imageParts := part.DepthMatchAll(imageMatcher)
	if len(imageParts) == 0 {
		errs = append(errs, fmt.Errorf("Keine Bilddatei in der E-Mail gefunden."))
		return nil, nil, "", errs
	}
	if len(imageParts) > 1 {
		errs = append(errs, fmt.Errorf("Mehrere Bilder gefunden. Die E-Mail darf maximal ein Bild entalten."))
		return nil, nil, "", errs
	}

	var err error
	if len(imageParts) == 1 {
		image, err = ioutil.ReadAll(imageParts[0])
		if err != nil {
			errs = append(errs, fmt.Errorf("Die Bilddatei kann nicht gelesen werden: %s", err))
		}
		filename = imageParts[0].FileName
	}
	imageObject, err := imaging.Decode(bytes.NewReader(image))
	if err != nil {
		errs = append(errs, fmt.Errorf("Die Bilddatei kann nicht gelesen werden: %s", err))
		return nil, nil, "", errs
	}

	thumbnailObject := imaging.Fill(imageObject, 250, 200, imaging.Center, imaging.Lanczos)
	buf := bytes.NewBuffer(make([]byte, 0))
	err = imaging.Encode(buf, thumbnailObject, imaging.JPEG)
	if err != nil {
		errs = append(errs, fmt.Errorf("Thumbnail kann aus dem Bild nicht erstellt werden: %s", err))
	}
	if len(errs) > 0 {
		return nil, nil, "", errs
	}
	return image, buf.Bytes(), filename, nil
}

func parseMail(mail *enmime.Envelope) (*mail.Address, string, string, []byte, string, []byte, []error) {
	errs := make([]error, 0)
	address, err := parseAuthor(mail.Root.Header.Get("from"))
	if err != nil {
		errs = append(errs, err...)
	}
	subject, err := parseSubject(mail.Root.Header.Get("subject"))
	if err != nil {
		errs = append(errs, err...)
	}
	text, err := parseText(mail.Text)
	if err != nil {
		errs = append(errs, err...)
	}

	image, thumbnail, imageName, err := parseAttachments(mail.Root)
	if err != nil {
		errs = append(errs, err...)
	}

	if len(errs) > 0 {
		return address, "", "", nil, "", nil, errs
	}
	return address, subject, text, image, imageName, thumbnail, nil
}

// TODO: Bei allen fatal fehlern stattdessen eine antwort zurückschicken
func readMail(io.Reader) *Entry {
	raw := new(bytes.Buffer)
	mail, err := enmime.ReadEnvelope(io.TeeReader(os.Stdin, raw))
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	address, subject, text, image, imageName, thumbnail, errs := parseMail(mail)
	if errs != nil {
		response := "Die von dir gesendete E-Mail kann nicht gelesen werden:\n"
		for _, err := range errs {
			response += err.Error() + "\n"
		}
		err = sendMail(
			address.Address,
			"Re: "+mail.Root.Header.Get("subject"),
			response,
		)
		if err != nil {
			log.Fatalf("Can not send response mail: %s", err)
		}
		return nil
	}
	return &Entry{raw, address, subject, text, image, imageName, thumbnail}
}

func insert(in io.Reader) {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	entry := readMail(in)
	if entry == nil {
		return
	}
	_, token, err := postEntry(entry)
	if err != nil {
		log.Fatal(err)
	}
	sendMail(
		entry.address.Address,
		"Re: "+entry.subject,
		fmt.Sprintf(`Hallo,
dein Bild wurde erfolgreich gespeichert. In den nächsten 24 Stunden kannst
du es über den Link:
%s/delete/%s
wieder löschen.`, baseURL, token),
	)
}
