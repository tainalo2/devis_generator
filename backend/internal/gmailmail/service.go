// Package gmailmail implements delegated Google OAuth and Gmail API sending.
package gmailmail

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

const (
	tokenReference = "default"
	gmailSendScope = "https://www.googleapis.com/auth/gmail.send"
	stateTTL       = 10 * time.Minute
)

var (
	ErrInvalidState = errors.New("invalid or expired OAuth state")
	ErrNotConnected = errors.New("Gmail account is not connected")
)

type Config struct {
	ClientID, ClientSecret, RedirectURL  string
	AuthorizeURL, TokenURL, GmailBaseURL string
	StateKey                             []byte
}

type secretVault interface {
	Put(context.Context, paymentsecrets.Actor, uuid.UUID, paymentsecrets.Kind, string, []byte) (paymentsecrets.Metadata, error)
	Reveal(context.Context, paymentsecrets.Actor, uuid.UUID, paymentsecrets.Kind, string) ([]byte, paymentsecrets.Metadata, error)
	Delete(context.Context, paymentsecrets.Actor, uuid.UUID, paymentsecrets.Kind, string) error
}

type Service struct {
	config Config
	vault  secretVault
	client *http.Client
	now    func() time.Time
}

type Message struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

func New(config Config, vault secretVault, client *http.Client) (*Service, error) {
	if config.ClientID == "" || config.ClientSecret == "" || config.RedirectURL == "" || len(config.StateKey) < 32 || vault == nil {
		return nil, errors.New("incomplete Google OAuth configuration")
	}
	if config.AuthorizeURL == "" {
		config.AuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	}
	if config.TokenURL == "" {
		config.TokenURL = "https://oauth2.googleapis.com/token"
	}
	if config.GmailBaseURL == "" {
		config.GmailBaseURL = "https://gmail.googleapis.com"
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Service{config: config, vault: vault, client: client, now: time.Now}, nil
}

func (s *Service) AuthorizationURL(ownerID uuid.UUID) (string, error) {
	if ownerID == uuid.Nil {
		return "", ErrInvalidState
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	payload := ownerID.String() + "." + strconv.FormatInt(s.now().Add(stateTTL).Unix(), 10) + "." + base64.RawURLEncoding.EncodeToString(nonce)
	state := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(s.sign(payload))
	q := url.Values{
		"access_type": {"offline"}, "client_id": {s.config.ClientID}, "include_granted_scopes": {"false"},
		"prompt": {"consent"}, "redirect_uri": {s.config.RedirectURL}, "response_type": {"code"},
		"scope": {gmailSendScope}, "state": {state},
	}
	return s.config.AuthorizeURL + "?" + q.Encode(), nil
}

func (s *Service) ValidateState(state string, ownerID uuid.UUID) error {
	parts := strings.Split(state, ".")
	if len(parts) != 2 {
		return ErrInvalidState
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return ErrInvalidState
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(signature, s.sign(string(payload))) {
		return ErrInvalidState
	}
	payloadParts := strings.Split(string(payload), ".")
	if len(payloadParts) != 3 || payloadParts[0] != ownerID.String() {
		return ErrInvalidState
	}
	expires, err := strconv.ParseInt(payloadParts[1], 10, 64)
	if err != nil || s.now().Unix() > expires {
		return ErrInvalidState
	}
	return nil
}

func (s *Service) sign(payload string) []byte {
	mac := hmac.New(sha256.New, s.config.StateKey)
	mac.Write([]byte(payload))
	return mac.Sum(nil)
}

func (s *Service) HandleCallback(ctx context.Context, actor paymentsecrets.Actor, ownerID uuid.UUID, state, code string) error {
	if err := s.ValidateState(state, ownerID); err != nil {
		return err
	}
	if strings.TrimSpace(code) == "" {
		return errors.New("authorization code is required")
	}
	token, err := s.exchange(ctx, url.Values{
		"client_id": {s.config.ClientID}, "client_secret": {s.config.ClientSecret},
		"code": {code}, "grant_type": {"authorization_code"}, "redirect_uri": {s.config.RedirectURL},
	})
	if err != nil {
		return err
	}
	if token.RefreshToken == "" {
		return errors.New("Google token response omitted refresh token")
	}
	_, err = s.vault.Put(ctx, actor, ownerID, paymentsecrets.KindOAuthGmailToken, tokenReference, []byte(token.RefreshToken))
	return err
}

func (s *Service) Send(ctx context.Context, actor paymentsecrets.Actor, ownerID uuid.UUID, message Message) error {
	rawMessage, err := encodeMessage(message)
	if err != nil {
		return err
	}
	refreshToken, _, err := s.vault.Reveal(ctx, actor, ownerID, paymentsecrets.KindOAuthGmailToken, tokenReference)
	if errors.Is(err, paymentsecrets.ErrNotFound) {
		return ErrNotConnected
	}
	if err != nil {
		return err
	}
	token, err := s.exchange(ctx, url.Values{
		"client_id": {s.config.ClientID}, "client_secret": {s.config.ClientSecret},
		"grant_type": {"refresh_token"}, "refresh_token": {string(refreshToken)},
	})
	if err != nil {
		return err
	}
	if token.RefreshToken != "" && token.RefreshToken != string(refreshToken) {
		if _, err := s.vault.Put(ctx, actor, ownerID, paymentsecrets.KindOAuthGmailToken, tokenReference, []byte(token.RefreshToken)); err != nil {
			return err
		}
	}
	payload, err := json.Marshal(map[string]string{"raw": base64.RawURLEncoding.EncodeToString(rawMessage)})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.config.GmailBaseURL, "/")+"/gmail/v1/users/me/messages/send", strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("Gmail messages.send returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return nil
}

func encodeMessage(message Message) ([]byte, error) {
	if len(message.To) == 0 || strings.TrimSpace(message.Subject) == "" {
		return nil, errors.New("recipient and subject are required")
	}
	if strings.ContainsAny(message.Subject, "\r\n") {
		return nil, errors.New("invalid subject")
	}
	recipients := make([]string, 0, len(message.To))
	for _, raw := range message.To {
		if strings.ContainsAny(raw, "\r\n") {
			return nil, errors.New("invalid recipient")
		}
		address, err := mail.ParseAddress(raw)
		if err != nil || address.Address == "" {
			return nil, errors.New("invalid recipient")
		}
		recipients = append(recipients, address.String())
	}
	var raw strings.Builder
	raw.WriteString("To: " + strings.Join(recipients, ", ") + "\r\n")
	raw.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", message.Subject) + "\r\n")
	raw.WriteString("MIME-Version: 1.0\r\n")
	raw.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	raw.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
	raw.WriteString(message.HTML)
	return []byte(raw.String()), nil
}

func (s *Service) Status(ctx context.Context, actor paymentsecrets.Actor, ownerID uuid.UUID) (bool, error) {
	_, _, err := s.vault.Reveal(ctx, actor, ownerID, paymentsecrets.KindOAuthGmailToken, tokenReference)
	if errors.Is(err, paymentsecrets.ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}

func (s *Service) Disconnect(ctx context.Context, actor paymentsecrets.Actor, ownerID uuid.UUID) error {
	err := s.vault.Delete(ctx, actor, ownerID, paymentsecrets.KindOAuthGmailToken, tokenReference)
	if errors.Is(err, paymentsecrets.ErrNotFound) {
		return nil
	}
	return err
}

func (s *Service) exchange(ctx context.Context, form url.Values) (tokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.client.Do(req)
	if err != nil {
		return tokenResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return tokenResponse{}, fmt.Errorf("Google token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var token tokenResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&token); err != nil {
		return tokenResponse{}, err
	}
	if token.AccessToken == "" {
		return tokenResponse{}, errors.New("Google token response omitted access token")
	}
	return token, nil
}
