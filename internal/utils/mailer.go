package utils

import (
	"fmt"
	"net/smtp"
)

type SMTPClient struct {
	Host     string
	User     string
	Password string
	From     string
}

func NewSMTPClient(host, user, pass, from string) *SMTPClient {
	return &SMTPClient{Host: host, User: user, Password: pass, From: from}
}

func (s *SMTPClient) Send(to, subject, body string) error {
	if s == nil || s.Host == "" || s.User == "" {
		return fmt.Errorf("smtp not configured")
	}
	addr := fmt.Sprintf("%s:%d", s.Host, 587)
	auth := smtp.PlainAuth("", s.User, s.Password, s.Host)
	msg := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"\r\n" + body + "\r\n")
	return smtp.SendMail(addr, auth, s.From, []string{to}, msg)
}
