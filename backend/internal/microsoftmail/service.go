// Package microsoftmail implements delegated Microsoft OAuth and Graph Mail.Send.
package microsoftmail

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
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

const (
	tokenReference = "default"
	mailSendScope  = "offline_access https://graph.microsoft.com/Mail.Send"
	stateTTL       = 10 * time.Minute
)

var (
	ErrInvalidState = errors.New("invalid or expired OAuth state")
	ErrNotConnected = errors.New("Microsoft mailbox is not connected")
)

type Config struct {
	ClientID, ClientSecret, RedirectURL  string
	AuthorizeURL, TokenURL, GraphBaseURL string
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
		return nil, errors.New("incomplete Microsoft OAuth configuration")
	}
	if config.AuthorizeURL == "" {
		config.AuthorizeURL = "https://login.microsoftonline.com/common/oauth2/v2.0/authorize"
	}
	if config.TokenURL == "" {
		config.TokenURL = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
	}
	if config.GraphBaseURL == "" {
		config.GraphBaseURL = "https://graph.microsoft.com"
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
	signature := s.sign(payload)
	state := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(signature)
	q := url.Values{
		"client_id": {s.config.ClientID}, "redirect_uri": {s.config.RedirectURL},
		"response_type": {"code"}, "response_mode": {"query"}, "scope": {mailSendScope}, "state": {state},
	}
	return s.config.AuthorizeURL + "?" + q.Encode(), nil
}

func (s *Service) ValidateState(state string, ownerID uuid.UUID) error {
	parts := strings.Split(state, ".")
	if len(parts) != 2 {
		return ErrInvalidState
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return ErrInvalidState
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(sig, s.sign(string(payloadBytes))) {
		return ErrInvalidState
	}
	payloadParts := strings.Split(string(payloadBytes), ".")
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
		"redirect_uri": {s.config.RedirectURL}, "grant_type": {"authorization_code"}, "code": {code}, "scope": {mailSendScope},
	})
	if err != nil {
		return err
	}
	if token.RefreshToken == "" {
		return errors.New("Microsoft token response omitted refresh token")
	}
	_, err = s.vault.Put(ctx, actor, ownerID, paymentsecrets.KindOAuthMicrosoftToken, tokenReference, []byte(token.RefreshToken))
	return err
}

func (s *Service) Send(ctx context.Context, actor paymentsecrets.Actor, ownerID uuid.UUID, message Message) error {
	if len(message.To) == 0 || strings.TrimSpace(message.Subject) == "" {
		return errors.New("recipient and subject are required")
	}
	refresh, _, err := s.vault.Reveal(ctx, actor, ownerID, paymentsecrets.KindOAuthMicrosoftToken, tokenReference)
	if errors.Is(err, paymentsecrets.ErrNotFound) {
		return ErrNotConnected
	}
	if err != nil {
		return err
	}
	token, err := s.exchange(ctx, url.Values{
		"client_id": {s.config.ClientID}, "client_secret": {s.config.ClientSecret},
		"grant_type": {"refresh_token"}, "refresh_token": {string(refresh)}, "scope": {mailSendScope},
	})
	if err != nil {
		return err
	}
	if token.RefreshToken != "" && token.RefreshToken != string(refresh) {
		if _, err := s.vault.Put(ctx, actor, ownerID, paymentsecrets.KindOAuthMicrosoftToken, tokenReference, []byte(token.RefreshToken)); err != nil {
			return err
		}
	}
	recipients := make([]map[string]any, 0, len(message.To))
	for _, address := range message.To {
		recipients = append(recipients, map[string]any{"emailAddress": map[string]string{"address": address}})
	}
	body := map[string]any{"message": map[string]any{"subject": message.Subject, "body": map[string]string{"contentType": "HTML", "content": message.HTML}, "toRecipients": recipients}, "saveToSentItems": true}
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.config.GraphBaseURL, "/")+"/v1.0/me/sendMail", strings.NewReader(string(encoded)))
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
	if resp.StatusCode != http.StatusAccepted {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("Graph sendMail returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return nil
}

func (s *Service) Status(ctx context.Context, actor paymentsecrets.Actor, ownerID uuid.UUID) (bool, error) {
	_, _, err := s.vault.Reveal(ctx, actor, ownerID, paymentsecrets.KindOAuthMicrosoftToken, tokenReference)
	if errors.Is(err, paymentsecrets.ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}

func (s *Service) Disconnect(ctx context.Context, actor paymentsecrets.Actor, ownerID uuid.UUID) error {
	err := s.vault.Delete(ctx, actor, ownerID, paymentsecrets.KindOAuthMicrosoftToken, tokenReference)
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
		return tokenResponse{}, fmt.Errorf("Microsoft token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var token tokenResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&token); err != nil {
		return tokenResponse{}, err
	}
	if token.AccessToken == "" {
		return tokenResponse{}, errors.New("Microsoft token response omitted access token")
	}
	return token, nil
}
