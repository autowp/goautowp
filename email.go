package goautowp

import (
	"fmt"
	"gopkg.in/gomail.v2"
	"strings"
)

type EmailSender interface {
	Send(from string, to []string, subject, body, replyTo string) error
}

type SmtpEmailSender struct {
	config SMTPConfig
}

type MockEmailSender struct {
	Body string
}

func (s *SmtpEmailSender) Send(from string, to []string, subject, body, replyTo string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)
	m.SetHeader("Reply-To", replyTo)

	d := gomail.NewDialer(s.config.Hostname, s.config.Port, s.config.Username, s.config.Password)

	return d.DialAndSend(m)
}

func (s *MockEmailSender) Send(from string, to []string, subject, body, _ string) error {
	fmt.Printf("Subject: %s\nFrom: %s\nTo: %s\n%s", subject, from, strings.Join(to, ", "), body)
	s.Body = body
	return nil
}
