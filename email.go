package goautowp

import "gopkg.in/gomail.v2"

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

func (s *MockEmailSender) Send(_ string, _ []string, _, body, _ string) error {
	s.Body = body
	return nil
}
