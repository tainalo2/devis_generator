// Package emailproviders implements email sending via OAuth2 and SMTP providers.
package emailproviders

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

// MicrosoftOAuthConfig holds the OAuth2 configuration for Microsoft Graph API.
type MicrosoftOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	TenantID     string // defaults to "common" for multi-tenant
}

// MicrosoftGraphClient sends emails via Microsoft Graph /me/sendMail endpoint.
type MicrosoftGraphClient struct {
	config *MicrosoftOAuthConfig
	vault  *paymentsecrets.Vault
}

// NewMicrosoftGraphClient creates a new Graph API client.
func NewMicrosoftGraphClient(config *MicrosoftOAuthConfig, vault *paymentsecrets.Vault) *MicrosoftGraphClient {
	if config.TenantID == "" {
		config.TenantID = "common"
	}
	return &MicrosoftGraphClient{
		config: config,
		vault:  vault,
	}
}

// OAuthStartURL returns the authorization URL for the OAuth2 flow.
func (c *MicrosoftGraphClient) OAuthStartURL(state string) string {
	return fmt.Sprintf(
		"https://login.microsoftonline.com/%s/oauth2/v2.0/authorize?"+
			"client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		c.config.TenantID,
		c.config.ClientID,
		strings.TrimSpace(c.config.RedirectURI),
		"https://graph.microsoft.com/.default offline_access",
		state,
	)
}

// OAuthTokenResponse represents the token response from Microsoft identity platform.
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Error        string `json:"error,omitempty"`
	ErrorDesc    string `json:"error_description,omitempty"`
}

// ExchangeCodeForToken exchanges an authorization code for an access token.
func (c *MicrosoftGraphClient) ExchangeCodeForToken(ctx context.Context, code string) (*OAuthTokenResponse, error) {
	body := strings.NewReader(fmt.Sprintf(
		"client_id=%s&client_secret=%s&code=%s&redirect_uri=%s&grant_type=authorization_code&scope=%s",
		c.config.ClientID,
		c.config.ClientSecret,
		code,
		c.config.RedirectURI,
		"https://graph.microsoft.com/.default offline_access",
	))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", c.config.TenantID),
		body)
	if err != nil {
		return nil, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp OAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("oauth error: %s (%s)", tokenResp.Error, tokenResp.ErrorDesc)
	}

	return &tokenResp, nil
}

// RefreshAccessToken uses a refresh token to obtain a new access token.
func (c *MicrosoftGraphClient) RefreshAccessToken(ctx context.Context, refreshToken string) (*OAuthTokenResponse, error) {
	body := strings.NewReader(fmt.Sprintf(
		"client_id=%s&client_secret=%s&refresh_token=%s&grant_type=refresh_token&scope=%s",
		c.config.ClientID,
		c.config.ClientSecret,
		refreshToken,
		"https://graph.microsoft.com/.default offline_access",
	))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", c.config.TenantID),
		body)
	if err != nil {
		return nil, fmt.Errorf("create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp OAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("oauth error: %s (%s)", tokenResp.Error, tokenResp.ErrorDesc)
	}

	return &tokenResp, nil
}

// SendMailRequest is the body for the /me/sendMail endpoint.
type SendMailRequest struct {
	Message struct {
		Subject      string `json:"subject"`
		Body         struct {
			ContentType string `json:"contentType"` // "HTML" or "text"
			Content     string `json:"content"`
		} `json:"body"`
		ToRecipients []struct {
			EmailAddress struct {
				Address string `json:"address"`
				Name    string `json:"name,omitempty"`
			} `json:"emailAddress"`
		} `json:"toRecipients"`
		CCRecipients []struct {
			EmailAddress struct {
				Address string `json:"address"`
				Name    string `json:"name,omitempty"`
			} `json:"emailAddress"`
		} `json:"ccRecipients,omitempty"`
		ReplyTo []struct {
			EmailAddress struct {
				Address string `json:"address"`
				Name    string `json:"name,omitempty"`
			} `json:"emailAddress"`
		} `json:"replyTo,omitempty"`
	} `json:"message"`
	SaveToSentItems bool `json:"saveToSentItems"`
}

// SendMail sends an email via Microsoft Graph /me/sendMail endpoint.
// accessToken is the OAuth2 access token; it must be refreshed if expired.
func (c *MicrosoftGraphClient) SendMail(ctx context.Context, accessToken string, req *SendMailRequest) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal sendMail request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://graph.microsoft.com/v1.0/me/sendMail",
		bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create sendMail request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("sendMail request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sendMail failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
