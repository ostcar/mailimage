package main

import (
	"bytes"
	"fmt"
	"net/mail"
	"os"
	"text/template"

	gomail "github.com/go-mail/mail"
)

var sendErrorTemplate = `Hallo{{ if .Name }} {{ .Name }}{{ end }},

dein Bild kann nicht gespeichert werden. Bitte behebe {{ $length := len .Messages }}{{ if gt $length 1 }}den folgenden{{ else }}die folgenden{{ end }} Fehler:
{{ range .Messages }}* {{ . }}{{ end }}

{{ .Regards }}
`

var sendSuccessTemplate = `Hallo {{ .Name }},

dein Bild wurde erfolgreich gespeichert. In den folgenden 24 Stunden kannst
du es mit einem klick auf den folgenden den Link:

{{ .RemoveLink }}

{{ .Regards }}
`

func sendMail(to string, subject string, text string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", fromAdress)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", text)

	if debug {
		m.WriteTo(os.Stdout)
		return nil
	}

	d := gomail.Dialer{Host: "localhost", Port: 25, StartTLSPolicy: gomail.NoStartTLS}
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("can not connect to mailserver: %s", err)
	}
	return nil
}

func sendError(to *mail.Address, subject string, messages []string) error {
	subject = "Re: " + subject

	tmpl, err := template.New("").Parse(sendErrorTemplate)
	if err != nil {
		return fmt.Errorf("can not parse template: %s", err)
	}

	var text bytes.Buffer
	err = tmpl.Execute(
		&text,
		struct {
			Name     string
			Messages []string
			Regards  string
		}{
			to.Name,
			messages,
			responseRegards,
		},
	)
	if err != nil {
		return fmt.Errorf("can not execute template: %s", err)
	}

	return sendMail(to.Address, subject, text.String())
}

func sendSuccess(to *mail.Address, subject, token string) error {
	subject = "Re: " + subject

	tmpl, err := template.New("").Parse(sendSuccessTemplate)
	if err != nil {
		return fmt.Errorf("can not parse template: %s", err)
	}

	var text bytes.Buffer
	err = tmpl.Execute(
		&text,
		struct {
			Name       string
			RemoveLink string
			Regards    string
		}{
			to.Name,
			fmt.Sprintf("%s/delete/%s", baseURL, token),
			responseRegards,
		},
	)
	if err != nil {
		return fmt.Errorf("can not execute template: %s", err)
	}

	return sendMail(to.Address, subject, text.String())
}
