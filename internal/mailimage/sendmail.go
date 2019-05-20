package mailimage

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"golang.org/x/xerrors"
	"gopkg.in/mail.v2"
)

var (
	mailErrorTmpl   = template.Must(template.New("mailError").Parse(sendErrorTemplate))
	mailSuccessTmpl = template.Must(template.New("mailSuccess").Parse(sendSuccessTemplate))
)

// sendMail sends an email to an receiver.
// If the environment varialbe DEBUG is set, prints the mail to stdout
func sendMail(to string, subject string, text string) error {
	m := mail.NewMessage()
	m.SetHeader("From", fromAdress)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", text)

	if os.Getenv("DEBUG") != "" {
		if _, err := m.WriteTo(os.Stdout); err != nil {
			return xerrors.Errorf("can not write mail to stdout: %w", err)
		}
		return nil
	}

	d := mail.Dialer{Host: smptHost, Port: smptPort, StartTLSPolicy: mail.NoStartTLS}
	if err := d.DialAndSend(m); err != nil {
		return xerrors.Errorf("can not send mail: %w", err)
	}
	return nil
}

// respondError response to an incomming mail with an error message.
func respondError(name, address, subject string, errs []error) error {
	var text bytes.Buffer
	err := mailErrorTmpl.Execute(
		&text,
		struct {
			Name    string
			Errors  []error
			Regards string
		}{
			name,
			errs,
			responseRegards,
		},
	)
	if err != nil {
		return xerrors.Errorf("can not render error template: %w", err)
	}

	return sendMail(address, "Re: "+subject, text.String())
}

// respondSuccess response to an incomming mail with an success message.
func respondSuccess(name, address, subject, token string) error {
	var text bytes.Buffer
	err := mailSuccessTmpl.Execute(
		&text,
		struct {
			Name       string
			ImageLink  string
			RemoveLink string
			Regards    string
		}{
			name,
			deleteRedirectURL,
			fmt.Sprintf("%s/delete/%s", baseURL, token),
			responseRegards,
		},
	)
	if err != nil {
		return xerrors.Errorf("can not execute template: %w", err)
	}

	return sendMail(address, "Re: "+subject, text.String())
}
