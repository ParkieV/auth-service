package email

import (
	"fmt"
	"net/smtp"

	"github.com/ParkieV/auth-service/internal/config"
)

type SMTPMailer struct {
	auth smtp.Auth
	host string
	from string
}

func NewSMTPMailer(cfg config.EmailConfig) *SMTPMailer {
	hostPort := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)
	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
	return &SMTPMailer{
		auth: auth,
		host: hostPort,
		from: cfg.From,
	}
}

func (m *SMTPMailer) Send(to, subject, htmlBody string) error {
	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\n"+
			"MIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		m.from, to, subject, htmlBody,
	))
	return smtp.SendMail(m.host, m.auth, m.from, []string{to}, msg)
}
