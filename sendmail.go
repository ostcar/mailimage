package main

import (
	"bytes"
	"fmt"
	"net/mail"
	"net/smtp"
	"text/template"
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
	if debug {
		fmt.Println(text)
		return nil
	}
	// Connect to the remote SMTP server.
	c, err := smtp.Dial("localhost:25")
	if err != nil {
		return fmt.Errorf("can not connect to smtp server: %s", err)
	}
	defer c.Close()

	// Set the sender and recipient.
	c.Mail(fromAdress)
	c.Rcpt(to)

	// Send the email body.
	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("can not send mail: %s", err)
	}
	defer wc.Close()

	buf := bytes.NewBufferString(text)
	if _, err = buf.WriteTo(wc); err != nil {
		return fmt.Errorf("can not send mail: %s", err)
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
