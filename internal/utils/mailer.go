package utils

import (
	"fmt"
	"net/smtp"
	"os"
)

type SMTPClient struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

func NewSMTPClient(host, user, pass, from string) *SMTPClient {
	port := os.Getenv("SMTP_PORT")
	return &SMTPClient{Host: host, Port: port, User: user, Password: pass, From: from}
}

func (s *SMTPClient) Send(to, subject, body string) error {
	if s == nil || s.Host == "" || s.User == "" || s.Port == "" || s.From == "" {
		return fmt.Errorf("smtp not configured")
	}
	addr := fmt.Sprintf("%s:%s", s.Host, s.Port)
	auth := smtp.PlainAuth("", s.User, s.Password, s.Host)
	msg := []byte("From: " + s.From + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"\r\n" + body + "\r\n")
	err := smtp.SendMail(addr, auth, s.From, []string{to}, msg)
	if err != nil {
		fmt.Printf("[SMTP ERROR] Failed to send mail: %v\n", err)
	}
	return err
}
