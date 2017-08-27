package main

import (
	"bytes"
	"fmt"
	"net/smtp"
)

func sendMail(to string, subject string, text string) error {
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
