package smtpconfig

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type ContextDialer interface {
	DialContext(context.Context, string, string) (net.Conn, error)
}

type SMTPTester struct {
	dialer    ContextDialer
	tlsConfig func(string) *tls.Config
}

func NewSMTPTester(dialer ContextDialer) *SMTPTester {
	if dialer == nil {
		dialer = &net.Dialer{Timeout: 10 * time.Second}
	}
	return &SMTPTester{
		dialer: dialer,
		tlsConfig: func(host string) *tls.Config {
			return &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
		},
	}
}

// Test connects, authenticates, and sends a test email to the configured From address.
func (t *SMTPTester) Test(ctx context.Context, config Config, password string) error {
	address := net.JoinHostPort(config.Host, fmt.Sprintf("%d", config.Port))
	conn, err := t.dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	var client *smtp.Client
	switch config.TLSMode {
	case TLSModeTLS:
		tlsConn := tls.Client(conn, t.tlsConfig(config.Host))
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			return err
		}
		client, err = smtp.NewClient(tlsConn, config.Host)
	case TLSModeSTARTTLS:
		client, err = smtp.NewClient(conn, config.Host)
		if err == nil {
			if ok, _ := client.Extension("STARTTLS"); !ok {
				client.Close()
				return fmt.Errorf("server does not support STARTTLS")
			}
			err = client.StartTLS(t.tlsConfig(config.Host))
		}
	default:
		return ErrInvalidConfig
	}
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Auth(smtp.PlainAuth("", config.Username, password, config.Host)); err != nil {
		return err
	}
	if err := client.Mail(config.FromAddress); err != nil {
		return err
	}
	if err := client.Rcpt(config.FromAddress); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	message := strings.Join([]string{
		"From: " + config.FromAddress,
		"To: " + config.FromAddress,
		"Subject: Devis Generator SMTP test",
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		"SMTP configuration is working.",
		"",
	}, "\r\n")
	if _, err := io.WriteString(writer, message); err != nil {
		writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
