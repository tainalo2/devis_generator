package smtpconfig

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/netip"
	"net/smtp"
	"strings"
	"time"
)

const smtpTestTimeout = 15 * time.Second

var ErrUnsafeDestination = fmt.Errorf("SMTP destination is not public")

type ContextDialer interface {
	DialContext(context.Context, string, string) (net.Conn, error)
}

type IPResolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

type SMTPTester struct {
	dialer    ContextDialer
	resolver  IPResolver
	tlsConfig func(string) *tls.Config
}

func NewSMTPTester(dialer ContextDialer) *SMTPTester {
	if dialer == nil {
		dialer = &net.Dialer{Timeout: 10 * time.Second}
	}
	return &SMTPTester{
		dialer:   dialer,
		resolver: net.DefaultResolver,
		tlsConfig: func(host string) *tls.Config {
			return &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
		},
	}
}

// Test connects, authenticates, and sends a test email to the configured From address.
func (t *SMTPTester) Test(ctx context.Context, config Config, password string) error {
	ctx, cancel := context.WithTimeout(ctx, smtpTestTimeout)
	defer cancel()
	addresses, err := t.resolver.LookupIPAddr(ctx, config.Host)
	if err != nil {
		return err
	}
	if len(addresses) == 0 {
		return ErrUnsafeDestination
	}
	for _, candidate := range addresses {
		if !isPublicIP(candidate.IP) {
			return fmt.Errorf("%w: %s", ErrUnsafeDestination, candidate.IP)
		}
	}
	// Dial the vetted literal, preventing a second DNS lookup from rebinding.
	address := net.JoinHostPort(addresses[0].IP.String(), fmt.Sprintf("%d", config.Port))
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

func isPublicIP(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	addr = addr.Unmap()
	if !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsMulticast() || addr.IsUnspecified() {
		return false
	}
	reserved := []netip.Prefix{
		netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"),
		netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"),
		netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("240.0.0.0/4"),
		netip.MustParsePrefix("100::/64"), netip.MustParsePrefix("2001:db8::/32"),
	}
	for _, prefix := range reserved {
		if prefix.Contains(addr) {
			return false
		}
	}
	return true
}
