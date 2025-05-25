package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"time"

	"github.com/ParkieV/auth-service/internal/config"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, html string) error
}

type SMTPMailer struct {
	addr   string
	from   string
	auth   smtp.Auth
	useTLS bool
	tlsCfg *tls.Config
	ttl    time.Duration
}

func NewSMTPMailer(cfg config.EmailConfig) *SMTPMailer {
	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	return &SMTPMailer{
		addr:   addr,
		from:   cfg.From,
		auth:   smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost),
		useTLS: cfg.UseTLS,
		tlsCfg: &tls.Config{ServerName: cfg.SMTPHost},
		ttl:    10 * time.Second,
	}
}

func (m *SMTPMailer) Send(ctx context.Context, to, subject, htmlBody string) error {
	msg := []byte(fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n\r\n%s",
		m.from, to, subject, htmlBody,
	))

	dialer := &net.Dialer{Timeout: m.ttl}
	conn, err := dialer.DialContext(ctx, "tcp", m.addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	var c *smtp.Client
	if m.useTLS {
		tlsConn := tls.Client(conn, m.tlsCfg)
		if err := tlsConn.Handshake(); err != nil {
			return err
		}
		c, err = smtp.NewClient(tlsConn, m.tlsCfg.ServerName)
	} else {
		c, err = smtp.NewClient(conn, m.tlsCfg.ServerName)
	}
	if err != nil {
		return err
	}
	defer c.Quit()

	if !m.useTLS {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(m.tlsCfg); err != nil {
				return err
			}
		}
	}

	if err := c.Auth(m.auth); err != nil {
		return err
	}

	if err := c.Mail(m.from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}

	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	return w.Close()
}
