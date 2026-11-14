package paymentbyo

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HTTPClient struct {
	BaseURL string
	Client  *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{BaseURL: strings.TrimRight(baseURL, "/"), Client: &http.Client{Timeout: 10 * time.Second}}
}

func (h *HTTPClient) do(ctx context.Context, method, path string, body []byte, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, method, h.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := h.Client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return b, resp.StatusCode, nil
}

// Stripe
type StripeAdapter struct{ HTTP *HTTPClient }

func (s *StripeAdapter) ValidateConfig(cfg Config) error {
	if cfg.Env != EnvSandbox && cfg.Env != EnvLive {
		return fmt.Errorf("invalid env")
	}
	if !strings.HasPrefix(cfg.StripeSecretKey, "sk_") {
		return fmt.Errorf("invalid secret key")
	}
	if cfg.StripeWebhookSecret == "" {
		return fmt.Errorf("missing webhook secret")
	}
	// env consistency
	if cfg.Env == EnvSandbox && !strings.HasPrefix(cfg.StripeSecretKey, "sk_test_") {
		return fmt.Errorf("sandbox requires test key")
	}
	if cfg.Env == EnvLive && !strings.HasPrefix(cfg.StripeSecretKey, "sk_live_") {
		return fmt.Errorf("live requires live key")
	}
	return nil
}
func (s *StripeAdapter) CreatePayment(ctx context.Context, cfg Config, req PaymentRequest) (PaymentSession, error) {
	if err := s.ValidateConfig(cfg); err != nil {
		return PaymentSession{}, err
	}
	b, _ := json.Marshal(map[string]any{"amount": req.Amount, "currency": req.Currency, "success_url": req.ReturnURL, "cancel_url": req.CancelURL, "metadata": map[string]string{"tenant_id": req.TenantID, "document_id": req.DocumentID}})
	resp, code, err := s.HTTP.do(ctx, "POST", "/v1/checkout/sessions", b, map[string]string{"Authorization": "Bearer " + cfg.StripeSecretKey, "Idempotency-Key": req.IdempotencyKey})
	if err != nil {
		return PaymentSession{}, err
	}
	if code >= 400 {
		return PaymentSession{}, fmt.Errorf("stripe %d: %s", code, string(resp))
	}
	var r struct{ ID, URL string }
	json.Unmarshal(resp, &r)
	return PaymentSession{Provider: ProviderStripe, Env: cfg.Env, ExternalID: r.ID, CheckoutURL: r.URL}, nil
}
func (s *StripeAdapter) VerifyWebhook(cfg Config, headers map[string]string, body []byte) (WebhookEvent, error) {
	sig := headers["Stripe-Signature"]
	if sig == "" || cfg.StripeWebhookSecret == "" {
		return WebhookEvent{}, fmt.Errorf("invalid signature")
	}
	parts := strings.Split(sig, ",")
	var timestamp, provided string
	for _, p := range parts {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			if kv[0] == "t" {
				timestamp = kv[1]
			}
			if kv[0] == "v1" {
				provided = kv[1]
			}
		}
	}
	mac := hmac.New(sha256.New, []byte(cfg.StripeWebhookSecret))
	mac.Write([]byte(timestamp + "." + string(body)))
	expected := hex.EncodeToString(mac.Sum(nil))
	if timestamp == "" || !hmac.Equal([]byte(expected), []byte(provided)) {
		return WebhookEvent{}, fmt.Errorf("invalid signature")
	}
	// parse body
	var ev struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Data struct {
			Object struct {
				ID       string            `json:"id"`
				Status   string            `json:"status"`
				Metadata map[string]string `json:"metadata"`
			} `json:"object"`
		} `json:"data"`
	}
	json.Unmarshal(body, &ev)
	status := "pending"
	if ev.Data.Object.Status == "paid" || ev.Type == "checkout.session.completed" {
		status = "paid"
	}
	docID := ""
	if ev.Data.Object.Metadata != nil {
		docID = ev.Data.Object.Metadata["document_id"]
	}
	return WebhookEvent{Provider: ProviderStripe, ExternalID: ev.Data.Object.ID, DocumentID: docID, Status: status, Raw: body}, nil
}
func (s *StripeAdapter) Revoke(cfg Config) error       { return nil }
func (s *StripeAdapter) Fingerprint(cfg Config) string { return "stripe:" + last4(cfg.StripeSecretKey) }

// Mollie
type MollieAdapter struct {
	HTTP            *HTTPClient
	WebhookVerifier func(Config, map[string]string, []byte) (WebhookEvent, error)
}

func (m *MollieAdapter) ValidateConfig(cfg Config) error {
	if cfg.Env != EnvSandbox && cfg.Env != EnvLive {
		return fmt.Errorf("invalid env")
	}
	if !strings.HasPrefix(cfg.MollieAPIKey, "test_") && !strings.HasPrefix(cfg.MollieAPIKey, "live_") {
		return fmt.Errorf("invalid api key")
	}
	if cfg.Env == EnvSandbox && !strings.HasPrefix(cfg.MollieAPIKey, "test_") {
		return fmt.Errorf("sandbox requires test key")
	}
	if cfg.Env == EnvLive && !strings.HasPrefix(cfg.MollieAPIKey, "live_") {
		return fmt.Errorf("live requires live key")
	}
	return nil
}
func (m *MollieAdapter) CreatePayment(ctx context.Context, cfg Config, req PaymentRequest) (PaymentSession, error) {
	if err := m.ValidateConfig(cfg); err != nil {
		return PaymentSession{}, err
	}
	b, _ := json.Marshal(map[string]any{"amount": map[string]any{"value": fmt.Sprintf("%.2f", float64(req.Amount)/100.0), "currency": req.Currency}, "redirectUrl": req.ReturnURL, "webhookUrl": req.CancelURL})
	resp, code, err := m.HTTP.do(ctx, "POST", "/v2/payments", b, map[string]string{"Authorization": "Bearer " + cfg.MollieAPIKey})
	if err != nil {
		return PaymentSession{}, err
	}
	if code >= 400 {
		return PaymentSession{}, fmt.Errorf("mollie %d", code)
	}
	var r struct {
		ID    string `json:"id"`
		Links struct {
			Checkout struct {
				Href string `json:"href"`
			} `json:"checkout"`
		} `json:"_links"`
	}
	json.Unmarshal(resp, &r)
	return PaymentSession{Provider: ProviderMollie, Env: cfg.Env, ExternalID: r.ID, CheckoutURL: r.Links.Checkout.Href}, nil
}
func (m *MollieAdapter) VerifyWebhook(cfg Config, headers map[string]string, body []byte) (WebhookEvent, error) {
	// Mollie authenticity requires retrieving the payment with the tenant API
	// key. Inject that verifier so tests never make real network calls.
	if m.WebhookVerifier == nil {
		return WebhookEvent{}, fmt.Errorf("mollie webhook verifier unavailable")
	}
	return m.WebhookVerifier(cfg, headers, body)
}
func (m *MollieAdapter) Revoke(cfg Config) error       { return nil }
func (m *MollieAdapter) Fingerprint(cfg Config) string { return "mollie:" + last4(cfg.MollieAPIKey) }

// PayPal
type PayPalAdapter struct {
	HTTP            *HTTPClient
	WebhookVerifier func(Config, map[string]string, []byte) (WebhookEvent, error)
}

func (p *PayPalAdapter) ValidateConfig(cfg Config) error {
	if cfg.Env != EnvSandbox && cfg.Env != EnvLive {
		return fmt.Errorf("invalid env")
	}
	if cfg.PayPalClientID == "" || cfg.PayPalClientSecret == "" || cfg.PayPalWebhookID == "" {
		return fmt.Errorf("missing creds")
	}
	return nil
}
func (p *PayPalAdapter) token(ctx context.Context, cfg Config) (string, error) {
	resp, code, err := p.HTTP.do(ctx, "POST", "/v1/oauth2/token", []byte("grant_type=client_credentials"), map[string]string{"Authorization": "Basic " + cfg.PayPalClientID + ":" + cfg.PayPalClientSecret, "Content-Type": "application/x-www-form-urlencoded"})
	if err != nil {
		return "", err
	}
	if code >= 400 {
		return "", fmt.Errorf("paypal auth %d", code)
	}
	var r struct {
		AccessToken string `json:"access_token"`
	}
	json.Unmarshal(resp, &r)
	return r.AccessToken, nil
}
func (p *PayPalAdapter) CreatePayment(ctx context.Context, cfg Config, req PaymentRequest) (PaymentSession, error) {
	if err := p.ValidateConfig(cfg); err != nil {
		return PaymentSession{}, err
	}
	tok, err := p.token(ctx, cfg)
	if err != nil {
		return PaymentSession{}, err
	}
	b, _ := json.Marshal(map[string]any{"intent": "CAPTURE", "purchase_units": []map[string]any{{"amount": map[string]any{"value": fmt.Sprintf("%.2f", float64(req.Amount)/100.0), "currency_code": req.Currency}}}})
	resp, code, err := p.HTTP.do(ctx, "POST", "/v2/checkout/orders", b, map[string]string{"Authorization": "Bearer " + tok})
	if err != nil {
		return PaymentSession{}, err
	}
	if code >= 400 {
		return PaymentSession{}, fmt.Errorf("paypal %d", code)
	}
	var r struct {
		ID    string                       `json:"id"`
		Links []struct{ Rel, Href string } `json:"links"`
	}
	json.Unmarshal(resp, &r)
	url := ""
	for _, l := range r.Links {
		if l.Rel == "approve" {
			url = l.Href
		}
	}
	return PaymentSession{Provider: ProviderPayPal, Env: cfg.Env, ExternalID: r.ID, CheckoutURL: url}, nil
}
func (p *PayPalAdapter) VerifyWebhook(cfg Config, headers map[string]string, body []byte) (WebhookEvent, error) {
	// PayPal's verify-webhook-signature API validates its certificate-backed
	// transmission headers. Keep the network verifier injected and fail closed.
	if p.WebhookVerifier == nil {
		return WebhookEvent{}, fmt.Errorf("paypal webhook verifier unavailable")
	}
	return p.WebhookVerifier(cfg, headers, body)
}
func (p *PayPalAdapter) Revoke(cfg Config) error       { return nil }
func (p *PayPalAdapter) Fingerprint(cfg Config) string { return "paypal:" + last4(cfg.PayPalClientID) }

func last4(s string) string {
	if len(s) <= 4 {
		return s
	}
	return s[len(s)-4:]
}
