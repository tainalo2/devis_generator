// Package smtpconfig manages per-user generic SMTP delivery settings.
package smtpconfig

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

const secretReference = "default"

type TLSMode string

const (
	TLSModeSTARTTLS TLSMode = "starttls"
	TLSModeTLS      TLSMode = "tls"
)

var (
	ErrInvalidConfig  = errors.New("invalid SMTP configuration")
	ErrNotFound       = errors.New("SMTP configuration not found")
	ErrSMTPTestFailed = errors.New("SMTP test failed")
)

type Input struct {
	Host        string  `json:"host"`
	Port        int     `json:"port"`
	Username    string  `json:"username"`
	Password    string  `json:"password"`
	FromAddress string  `json:"from_address"`
	TLSMode     TLSMode `json:"tls_mode"`
}

// Config deliberately has no password field and is safe to return from APIs.
type Config struct {
	OwnerID            uuid.UUID `json:"-"`
	Host               string    `json:"host"`
	Port               int       `json:"port"`
	Username           string    `json:"username"`
	FromAddress        string    `json:"from_address"`
	TLSMode            TLSMode   `json:"tls_mode"`
	PasswordConfigured bool      `json:"password_configured"`
	CreatedAt          time.Time `json:"created_at,omitempty"`
	UpdatedAt          time.Time `json:"updated_at,omitempty"`
}

type Store interface {
	Get(context.Context, uuid.UUID) (Config, error)
	Save(context.Context, paymentsecrets.Actor, Config, paymentsecrets.SealedSecret) (Config, error)
	Delete(context.Context, uuid.UUID) error
}

type Vault interface {
	Seal([]byte) (paymentsecrets.SealedSecret, error)
	Delete(context.Context, paymentsecrets.Actor, uuid.UUID, paymentsecrets.Kind, string) error
}

type Tester interface {
	Test(context.Context, Config, string) error
}

type Service struct {
	store  Store
	vault  Vault
	tester Tester
}

func NewService(store Store, vault Vault, tester Tester) *Service {
	return &Service{store: store, vault: vault, tester: tester}
}

func (s *Service) Save(ctx context.Context, actor paymentsecrets.Actor, ownerID uuid.UUID, input Input) (Config, error) {
	config, password, err := validate(ownerID, input)
	if err != nil {
		return Config{}, err
	}
	if err := s.tester.Test(ctx, config, password); err != nil {
		return Config{}, fmt.Errorf("%w: %v", ErrSMTPTestFailed, err)
	}
	sealed, err := s.vault.Seal([]byte(password))
	if err != nil {
		return Config{}, err
	}
	stored, err := s.store.Save(ctx, actor, config, sealed)
	if err != nil {
		return Config{}, err
	}
	stored.PasswordConfigured = true
	return stored, nil
}

func (s *Service) Get(ctx context.Context, ownerID uuid.UUID) (Config, error) {
	return s.store.Get(ctx, ownerID)
}

func (s *Service) Delete(ctx context.Context, actor paymentsecrets.Actor, ownerID uuid.UUID) error {
	if err := s.vault.Delete(ctx, actor, ownerID, paymentsecrets.KindSMTPPassword, secretReference); err != nil && !errors.Is(err, paymentsecrets.ErrNotFound) {
		return err
	}
	return s.store.Delete(ctx, ownerID)
}

func validate(ownerID uuid.UUID, input Input) (Config, string, error) {
	host := strings.TrimSpace(input.Host)
	username := strings.TrimSpace(input.Username)
	password := input.Password
	from := strings.TrimSpace(strings.ToLower(input.FromAddress))
	if ownerID == uuid.Nil || !validHost(host) || input.Port < 1 || input.Port > 65535 {
		return Config{}, "", ErrInvalidConfig
	}
	address, err := mail.ParseAddress(from)
	if err != nil || address.Address != from || utf8.RuneCountInString(from) > 254 {
		return Config{}, "", ErrInvalidConfig
	}
	if input.TLSMode != TLSModeSTARTTLS && input.TLSMode != TLSModeTLS {
		return Config{}, "", ErrInvalidConfig
	}
	if username == "" || password == "" || utf8.RuneCountInString(username) > 254 || utf8.RuneCountInString(password) > 1024 {
		return Config{}, "", ErrInvalidConfig
	}
	return Config{OwnerID: ownerID, Host: host, Port: input.Port, Username: username, FromAddress: from, TLSMode: input.TLSMode, PasswordConfigured: true}, password, nil
}

func validHost(host string) bool {
	if host == "" || len(host) > 253 || strings.ContainsAny(host, " \t\r\n:/") {
		return false
	}
	if net.ParseIP(host) != nil {
		return true
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' {
				return false
			}
		}
	}
	return true
}
