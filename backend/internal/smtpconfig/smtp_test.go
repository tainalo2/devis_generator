package smtpconfig

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

type dialFunc func(context.Context, string, string) (net.Conn, error)

func (f dialFunc) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return f(ctx, network, address)
}

type smtpServerResult struct {
	message    string
	tlsVersion uint16
	err        error
}

func TestSMTPTesterSendsTestMessageWithSTARTTLSAndTLS12Minimum(t *testing.T) {
	testSMTPTransport(t, TLSModeSTARTTLS)
}

func TestSMTPTesterSendsTestMessageWithImplicitTLSAndTLS12Minimum(t *testing.T) {
	testSMTPTransport(t, TLSModeTLS)
}

func testSMTPTransport(t *testing.T, mode TLSMode) {
	t.Helper()
	cert := localhostCertificate(t)
	results := make(chan smtpServerResult, 1)
	dialer := dialFunc(func(_ context.Context, network, address string) (net.Conn, error) {
		if network != "tcp" || address != "localhost:2525" {
			t.Fatalf("dial %s %s", network, address)
		}
		client, server := net.Pipe()
		go serveSMTP(server, mode, cert, results)
		return client, nil
	})
	tester := NewSMTPTester(dialer)
	tester.tlsConfig = func(host string) *tls.Config {
		return &tls.Config{ServerName: host, RootCAs: certPool(t, cert), MinVersion: tls.VersionTLS12}
	}
	config := Config{Host: "localhost", Port: 2525, Username: "mailer", FromAddress: "mailer@example.com", TLSMode: mode}

	if err := tester.Test(context.Background(), config, "secret"); err != nil {
		t.Fatal(err)
	}
	result := <-results
	if result.err != nil {
		t.Fatal(result.err)
	}
	if result.tlsVersion < tls.VersionTLS12 {
		t.Fatalf("TLS version=%x", result.tlsVersion)
	}
	if !strings.Contains(result.message, "Subject: Devis Generator SMTP test") || !strings.Contains(result.message, "SMTP configuration is working") {
		t.Fatalf("message=%q", result.message)
	}
}

func serveSMTP(conn net.Conn, mode TLSMode, cert tls.Certificate, results chan<- smtpServerResult) {
	defer conn.Close()
	result := smtpServerResult{}
	defer func() { results <- result }()
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	if mode == TLSModeTLS {
		tlsConn := tls.Server(conn, tlsCfg)
		if err := tlsConn.Handshake(); err != nil {
			result.err = err
			return
		}
		result.tlsVersion = tlsConn.ConnectionState().Version
		conn = tlsConn
	}
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	writeLine := func(line string) error {
		if _, err := fmt.Fprintf(w, "%s\r\n", line); err != nil {
			return err
		}
		return w.Flush()
	}
	if err := writeLine("220 localhost ESMTP"); err != nil {
		result.err = err
		return
	}
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			result.err = err
			return
		}
		command := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(command, "EHLO"):
			if mode == TLSModeSTARTTLS && result.tlsVersion == 0 {
				_, err = fmt.Fprint(w, "250-localhost\r\n250-STARTTLS\r\n250 AUTH PLAIN\r\n")
			} else {
				_, err = fmt.Fprint(w, "250-localhost\r\n250 AUTH PLAIN\r\n")
			}
			if err == nil {
				err = w.Flush()
			}
		case command == "STARTTLS":
			if err = writeLine("220 ready"); err == nil {
				tlsConn := tls.Server(conn, tlsCfg)
				err = tlsConn.Handshake()
				if err == nil {
					result.tlsVersion = tlsConn.ConnectionState().Version
					conn, r, w = tlsConn, bufio.NewReader(tlsConn), bufio.NewWriter(tlsConn)
				}
			}
		case strings.HasPrefix(command, "AUTH PLAIN"):
			err = writeLine("235 authenticated")
		case strings.HasPrefix(command, "MAIL FROM") || strings.HasPrefix(command, "RCPT TO"):
			err = writeLine("250 ok")
		case command == "DATA":
			if err = writeLine("354 send data"); err == nil {
				var lines []string
				for {
					line, err = r.ReadString('\n')
					if err != nil || line == ".\r\n" {
						break
					}
					lines = append(lines, line)
				}
				result.message = strings.Join(lines, "")
				if err == nil {
					err = writeLine("250 queued")
				}
			}
		case command == "QUIT":
			if err = writeLine("221 bye"); err != nil {
				result.err = err
				return
			}
			_ = conn.SetReadDeadline(time.Now().Add(time.Second))
			_, _ = r.ReadByte()
			return
		default:
			err = writeLine("500 unexpected")
		}
		if err != nil {
			result.err = err
			return
		}
	}
}

func localhostCertificate(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"}, DNSNames: []string{"localhost"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: &template}
}

func certPool(t *testing.T, cert tls.Certificate) *x509.CertPool {
	t.Helper()
	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(parsed)
	return pool
}
