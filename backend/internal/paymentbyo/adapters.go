package paymentbyo

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
	b, err := io.ReadAll(resp.Body)
	return b, resp.StatusCode, err
}

type StripeAdapter struct {
	HTTP *HTTPClient
	Now  func() time.Time
}

func (s *StripeAdapter) ValidateConfig(c Config) error {
	if c.Env != EnvSandbox && c.Env != EnvLive {
		return fmt.Errorf("invalid env")
	}
	if c.StripeWebhookSecret == "" {
		return fmt.Errorf("missing webhook secret")
	}
	if c.Env == EnvSandbox && !strings.HasPrefix(c.StripeSecretKey, "sk_test_") {
		return fmt.Errorf("sandbox requires test key")
	}
	if c.Env == EnvLive && !strings.HasPrefix(c.StripeSecretKey, "sk_live_") {
		return fmt.Errorf("live requires live key")
	}
	return nil
}
func (s *StripeAdapter) CreatePayment(ctx context.Context, c Config, r PaymentRequest) (PaymentSession, error) {
	if err := s.ValidateConfig(c); err != nil {
		return PaymentSession{}, err
	}
	v := url.Values{"mode": {"payment"}, "success_url": {r.ReturnURL}, "cancel_url": {r.CancelURL}, "line_items[0][quantity]": {"1"}, "line_items[0][price_data][unit_amount]": {strconv.FormatInt(r.Amount, 10)}, "line_items[0][price_data][currency]": {strings.ToLower(r.Currency)}, "line_items[0][price_data][product_data][name]": {"Invoice " + r.DocumentID}, "metadata[tenant_id]": {r.TenantID}, "metadata[document_id]": {r.DocumentID}}
	b, code, err := s.HTTP.do(ctx, "POST", "/v1/checkout/sessions", []byte(v.Encode()), map[string]string{"Authorization": "Bearer " + c.StripeSecretKey, "Content-Type": "application/x-www-form-urlencoded", "Idempotency-Key": r.IdempotencyKey})
	if err != nil {
		return PaymentSession{}, err
	}
	if code >= 400 {
		return PaymentSession{}, fmt.Errorf("stripe %d: %s", code, b)
	}
	var out struct{ ID, URL string }
	if err = json.Unmarshal(b, &out); err != nil {
		return PaymentSession{}, err
	}
	return PaymentSession{Provider: ProviderStripe, Env: c.Env, ExternalID: out.ID, CheckoutURL: out.URL}, nil
}
func (s *StripeAdapter) VerifyWebhook(c Config, h map[string]string, b []byte) (WebhookEvent, error) {
	var ts int64
	var provided string
	for _, p := range strings.Split(h["Stripe-Signature"], ",") {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) == 2 {
			if kv[0] == "t" {
				ts, _ = strconv.ParseInt(kv[1], 10, 64)
			}
			if kv[0] == "v1" {
				provided = kv[1]
			}
		}
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	if ts == 0 || time.Duration(abs(now().Unix()-ts))*time.Second > 5*time.Minute {
		return WebhookEvent{}, fmt.Errorf("invalid timestamp")
	}
	mac := hmac.New(sha256.New, []byte(c.StripeWebhookSecret))
	fmt.Fprintf(mac, "%d.%s", ts, b)
	expected, _ := hex.DecodeString(hex.EncodeToString(mac.Sum(nil)))
	got, err := hex.DecodeString(provided)
	if err != nil || !hmac.Equal(expected, got) {
		return WebhookEvent{}, fmt.Errorf("invalid signature")
	}
	var x struct {
		ID, Type string
		Data     struct {
			Object struct {
				ID            string            `json:"id"`
				PaymentStatus string            `json:"payment_status"`
				Metadata      map[string]string `json:"metadata"`
			} `json:"object"`
		} `json:"data"`
	}
	if err = json.Unmarshal(b, &x); err != nil {
		return WebhookEvent{}, err
	}
	status := "pending"
	if x.Type == "checkout.session.completed" && x.Data.Object.PaymentStatus == "paid" {
		status = "paid"
	}
	return WebhookEvent{Provider: ProviderStripe, EventID: x.ID, ExternalID: x.Data.Object.ID, TenantID: x.Data.Object.Metadata["tenant_id"], DocumentID: x.Data.Object.Metadata["document_id"], Status: status, Raw: b}, nil
}
func (s *StripeAdapter) Revoke(Config) error         { return nil }
func (s *StripeAdapter) Fingerprint(c Config) string { return "stripe:" + last4(c.StripeSecretKey) }

type MollieAdapter struct {
	HTTP       *HTTPClient
	WebhookURL string
}

func (m *MollieAdapter) ValidateConfig(c Config) error {
	if c.Env != EnvSandbox && c.Env != EnvLive {
		return fmt.Errorf("invalid env")
	}
	if c.Env == EnvSandbox && !strings.HasPrefix(c.MollieAPIKey, "test_") {
		return fmt.Errorf("sandbox requires test key")
	}
	if c.Env == EnvLive && !strings.HasPrefix(c.MollieAPIKey, "live_") {
		return fmt.Errorf("live requires live key")
	}
	return nil
}
func (m *MollieAdapter) CreatePayment(ctx context.Context, c Config, r PaymentRequest) (PaymentSession, error) {
	if err := m.ValidateConfig(c); err != nil {
		return PaymentSession{}, err
	}
	webhookURL, err := url.Parse(m.WebhookURL)
	if err != nil || webhookURL.Scheme != "https" || webhookURL.Host == "" {
		return PaymentSession{}, fmt.Errorf("public mollie webhook URL is required")
	}
	payload := map[string]any{"amount": map[string]string{"value": fmt.Sprintf("%.2f", float64(r.Amount)/100), "currency": r.Currency}, "redirectUrl": r.ReturnURL, "webhookUrl": m.WebhookURL, "metadata": map[string]string{"tenant_id": r.TenantID, "document_id": r.DocumentID}}
	b, _ := json.Marshal(payload)
	out, code, err := m.HTTP.do(ctx, "POST", "/v2/payments", b, map[string]string{"Authorization": "Bearer " + c.MollieAPIKey, "Content-Type": "application/json", "Idempotency-Key": r.IdempotencyKey})
	if err != nil || code >= 400 {
		if err == nil {
			err = fmt.Errorf("mollie %d", code)
		}
		return PaymentSession{}, err
	}
	var x struct {
		ID    string
		Links struct {
			Checkout struct{ Href string } `json:"checkout"`
		} `json:"_links"`
	}
	if err = json.Unmarshal(out, &x); err != nil {
		return PaymentSession{}, err
	}
	return PaymentSession{Provider: ProviderMollie, Env: c.Env, ExternalID: x.ID, CheckoutURL: x.Links.Checkout.Href}, nil
}
func (m *MollieAdapter) VerifyWebhook(c Config, _ map[string]string, b []byte) (WebhookEvent, error) {
	if m.HTTP == nil {
		return WebhookEvent{}, fmt.Errorf("mollie webhook verifier unavailable")
	}
	v, err := url.ParseQuery(string(b))
	if err != nil || v.Get("id") == "" {
		return WebhookEvent{}, fmt.Errorf("missing mollie payment id")
	}
	id := v.Get("id")
	out, code, err := m.HTTP.do(context.Background(), "GET", "/v2/payments/"+url.PathEscape(id), nil, map[string]string{"Authorization": "Bearer " + c.MollieAPIKey})
	if err != nil || code >= 400 {
		if err == nil {
			err = fmt.Errorf("mollie verify %d", code)
		}
		return WebhookEvent{}, err
	}
	var x struct {
		ID, Status string
		Metadata   map[string]string
	}
	if err = json.Unmarshal(out, &x); err != nil {
		return WebhookEvent{}, err
	}
	status := "pending"
	if x.Status == "paid" {
		status = "paid"
	} else if x.Status == "failed" || x.Status == "canceled" || x.Status == "expired" {
		status = "failed"
	}
	return WebhookEvent{Provider: ProviderMollie, EventID: "payment:" + x.ID + ":" + x.Status, ExternalID: x.ID, TenantID: x.Metadata["tenant_id"], DocumentID: x.Metadata["document_id"], Status: status, Raw: b}, nil
}
func (m *MollieAdapter) Revoke(Config) error         { return nil }
func (m *MollieAdapter) Fingerprint(c Config) string { return "mollie:" + last4(c.MollieAPIKey) }

type PayPalAdapter struct{ SandboxHTTP, LiveHTTP *HTTPClient }

func NewPayPalAdapter(sandbox, live *HTTPClient) *PayPalAdapter { return &PayPalAdapter{sandbox, live} }
func (p *PayPalAdapter) client(e Env) *HTTPClient {
	if e == EnvLive {
		return p.LiveHTTP
	}
	return p.SandboxHTTP
}
func (p *PayPalAdapter) ValidateConfig(c Config) error {
	if c.Env != EnvSandbox && c.Env != EnvLive {
		return fmt.Errorf("invalid env")
	}
	if c.PayPalClientID == "" || c.PayPalClientSecret == "" || c.PayPalWebhookID == "" {
		return fmt.Errorf("missing creds")
	}
	return nil
}
func (p *PayPalAdapter) token(ctx context.Context, c Config) (string, error) {
	auth := base64.StdEncoding.EncodeToString([]byte(c.PayPalClientID + ":" + c.PayPalClientSecret))
	b, code, err := p.client(c.Env).do(ctx, "POST", "/v1/oauth2/token", []byte("grant_type=client_credentials"), map[string]string{"Authorization": "Basic " + auth, "Content-Type": "application/x-www-form-urlencoded"})
	if err != nil || code >= 400 {
		if err == nil {
			err = fmt.Errorf("paypal auth %d", code)
		}
		return "", err
	}
	var x struct {
		AccessToken string `json:"access_token"`
	}
	err = json.Unmarshal(b, &x)
	return x.AccessToken, err
}
func (p *PayPalAdapter) CreatePayment(ctx context.Context, c Config, r PaymentRequest) (PaymentSession, error) {
	if err := p.ValidateConfig(c); err != nil {
		return PaymentSession{}, err
	}
	tok, err := p.token(ctx, c)
	if err != nil {
		return PaymentSession{}, err
	}
	payload := map[string]any{"intent": "CAPTURE", "purchase_units": []any{map[string]any{"reference_id": r.DocumentID, "custom_id": r.TenantID, "amount": map[string]string{"value": fmt.Sprintf("%.2f", float64(r.Amount)/100), "currency_code": r.Currency}}}, "application_context": map[string]string{"return_url": r.ReturnURL, "cancel_url": r.CancelURL}}
	b, _ := json.Marshal(payload)
	out, code, err := p.client(c.Env).do(ctx, "POST", "/v2/checkout/orders", b, map[string]string{"Authorization": "Bearer " + tok, "Content-Type": "application/json", "PayPal-Request-Id": r.IdempotencyKey})
	if err != nil || code >= 400 {
		if err == nil {
			err = fmt.Errorf("paypal %d", code)
		}
		return PaymentSession{}, err
	}
	var x struct {
		ID    string
		Links []struct{ Rel, Href string }
	}
	if err = json.Unmarshal(out, &x); err != nil {
		return PaymentSession{}, err
	}
	u := ""
	for _, l := range x.Links {
		if l.Rel == "approve" {
			u = l.Href
		}
	}
	return PaymentSession{Provider: ProviderPayPal, Env: c.Env, ExternalID: x.ID, CheckoutURL: u}, nil
}
func (p *PayPalAdapter) VerifyWebhook(c Config, h map[string]string, b []byte) (WebhookEvent, error) {
	required := []string{"Paypal-Transmission-Id", "Paypal-Transmission-Time", "Paypal-Cert-Url", "Paypal-Auth-Algo", "Paypal-Transmission-Sig"}
	for _, k := range required {
		if h[k] == "" {
			return WebhookEvent{}, fmt.Errorf("missing %s", k)
		}
	}
	tok, err := p.token(context.Background(), c)
	if err != nil {
		return WebhookEvent{}, err
	}
	var event any
	if err = json.Unmarshal(b, &event); err != nil {
		return WebhookEvent{}, err
	}
	verify := map[string]any{"auth_algo": h["Paypal-Auth-Algo"], "cert_url": h["Paypal-Cert-Url"], "transmission_id": h["Paypal-Transmission-Id"], "transmission_sig": h["Paypal-Transmission-Sig"], "transmission_time": h["Paypal-Transmission-Time"], "webhook_id": c.PayPalWebhookID, "webhook_event": event}
	vb, _ := json.Marshal(verify)
	out, code, err := p.client(c.Env).do(context.Background(), "POST", "/v1/notifications/verify-webhook-signature", vb, map[string]string{"Authorization": "Bearer " + tok, "Content-Type": "application/json"})
	if err != nil || code >= 400 {
		if err == nil {
			err = fmt.Errorf("paypal verify %d", code)
		}
		return WebhookEvent{}, err
	}
	var vr struct {
		VerificationStatus string `json:"verification_status"`
	}
	if json.Unmarshal(out, &vr) != nil || vr.VerificationStatus != "SUCCESS" {
		return WebhookEvent{}, fmt.Errorf("invalid paypal signature")
	}
	var x struct {
		ID        string `json:"id"`
		EventType string `json:"event_type"`
		Resource  struct {
			ID            string `json:"id"`
			Status        string `json:"status"`
			CustomID      string `json:"custom_id"`
			PurchaseUnits []struct {
				ReferenceID string `json:"reference_id"`
				CustomID    string `json:"custom_id"`
			} `json:"purchase_units"`
			SupplementaryData struct {
				RelatedIDs struct {
					OrderID string `json:"order_id"`
				} `json:"related_ids"`
			} `json:"supplementary_data"`
		} `json:"resource"`
	}
	if err = json.Unmarshal(b, &x); err != nil {
		return WebhookEvent{}, err
	}
	status := "pending"
	if x.EventType == "CHECKOUT.ORDER.COMPLETED" || x.EventType == "PAYMENT.CAPTURE.COMPLETED" {
		status = "paid"
	} else if x.EventType == "PAYMENT.CAPTURE.DENIED" || x.EventType == "CHECKOUT.PAYMENT-APPROVAL.REVERSED" {
		status = "failed"
	}
	externalID := x.Resource.ID
	if x.EventType == "PAYMENT.CAPTURE.COMPLETED" || x.EventType == "PAYMENT.CAPTURE.DENIED" {
		externalID = x.Resource.SupplementaryData.RelatedIDs.OrderID
	}
	doc, tenant := "", x.Resource.CustomID
	if len(x.Resource.PurchaseUnits) > 0 {
		doc = x.Resource.PurchaseUnits[0].ReferenceID
		if tenant == "" {
			tenant = x.Resource.PurchaseUnits[0].CustomID
		}
	}
	return WebhookEvent{Provider: ProviderPayPal, EventID: x.ID, ExternalID: externalID, TenantID: tenant, DocumentID: doc, Status: status, Raw: b}, nil
}
func (p *PayPalAdapter) Revoke(Config) error         { return nil }
func (p *PayPalAdapter) Fingerprint(c Config) string { return "paypal:" + last4(c.PayPalClientID) }
func last4(s string) string {
	if len(s) <= 4 {
		return s
	}
	return s[len(s)-4:]
}
func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
