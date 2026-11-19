package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tainalo2/devis_generator/backend/internal/config"
	"github.com/tainalo2/devis_generator/backend/internal/gmailmail"
	"github.com/tainalo2/devis_generator/backend/internal/microsoftmail"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
	"github.com/tainalo2/devis_generator/backend/internal/reminders"
	"github.com/tainalo2/devis_generator/backend/internal/smtpconfig"
)

// emailServices groups the services behind the reminder routes. Reminders and SMTP are always
// wired; Gmail and Microsoft only when their OAuth client is configured (nil otherwise, and the
// routes then answer 503 "indisponible" instead of panicking).
type emailServices struct {
	Reminders *reminders.Service
	SMTP      *smtpconfig.Service
	Gmail     *gmailmail.Service
	Microsoft *microsoftmail.Service
}

// oauthStateKey derives, per provider, the key that signs the OAuth state from the session
// secret: no new secret to provision, and a state signed for one provider is invalid for the other.
func oauthStateKey(sessionSecret, provider string) []byte {
	sum := sha256.Sum256([]byte("oauth-state:" + provider + ":" + sessionSecret))
	return sum[:]
}

func buildEmailServices(pool *pgxpool.Pool, vault *paymentsecrets.Vault, cfg config.Config) (emailServices, error) {
	var out emailServices
	smtpTester := smtpconfig.NewSMTPTester(&net.Dialer{Timeout: 10 * time.Second})
	out.SMTP = smtpconfig.NewService(smtpconfig.NewPGStore(pool), vault, smtpTester)

	needsKey := cfg.GoogleOAuthClientID != "" || cfg.MicrosoftOAuthClientID != ""
	if needsKey && cfg.SessionSecret == "" {
		return emailServices{}, errors.New("SESSION_SECRET is required to sign the OAuth state")
	}
	if cfg.GoogleOAuthClientID != "" {
		svc, err := gmailmail.New(gmailmail.Config{
			ClientID: cfg.GoogleOAuthClientID, ClientSecret: cfg.GoogleOAuthClientSecret, RedirectURL: cfg.GoogleOAuthRedirectURL,
			StateKey: oauthStateKey(cfg.SessionSecret, "gmail"),
		}, vault, nil)
		if err != nil {
			return emailServices{}, err
		}
		out.Gmail = svc
	}
	if cfg.MicrosoftOAuthClientID != "" {
		svc, err := microsoftmail.New(microsoftmail.Config{
			ClientID: cfg.MicrosoftOAuthClientID, ClientSecret: cfg.MicrosoftOAuthClientSecret, RedirectURL: cfg.MicrosoftOAuthRedirectURL,
			StateKey: oauthStateKey(cfg.SessionSecret, "microsoft"),
		}, vault, nil)
		if err != nil {
			return emailServices{}, err
		}
		out.Microsoft = svc
	}

	store := reminders.NewPGStore(pool)
	var gmailSend func(context.Context, paymentsecrets.Actor, uuid.UUID, gmailmail.Message) error
	if out.Gmail != nil {
		gmailSend = out.Gmail.Send
	}
	var microsoftSend func(context.Context, paymentsecrets.Actor, uuid.UUID, microsoftmail.Message) error
	if out.Microsoft != nil {
		microsoftSend = out.Microsoft.Send
	}
	router := reminders.NewProviderRouter(store, reminderSenders(gmailSend, microsoftSend, out.SMTP.SendMail))
	out.Reminders = reminders.NewService(store, router)
	return out, nil
}

// reminderSenders maps each delivery mode to the sender of the owner's own account. A nil
// function leaves its mode out: the router then refuses with ErrDeliveryMode, never falls back
// to another provider.
func reminderSenders(
	gmailSend func(context.Context, paymentsecrets.Actor, uuid.UUID, gmailmail.Message) error,
	microsoftSend func(context.Context, paymentsecrets.Actor, uuid.UUID, microsoftmail.Message) error,
	smtpSend func(ctx context.Context, actor paymentsecrets.Actor, owner uuid.UUID, to, subject, html string) error,
) map[reminders.DeliveryMode]reminders.Sender {
	senders := map[reminders.DeliveryMode]reminders.Sender{}
	if gmailSend != nil {
		senders[reminders.DeliveryModeGmail] = reminders.SenderFunc(func(ctx context.Context, owner uuid.UUID, m reminders.Message) error {
			return gmailSend(ctx, paymentsecrets.Actor{UserID: owner}, owner, gmailmail.Message{To: []string{m.To}, Subject: m.Subject, HTML: m.HTML})
		})
	}
	if microsoftSend != nil {
		senders[reminders.DeliveryModeMicrosoft] = reminders.SenderFunc(func(ctx context.Context, owner uuid.UUID, m reminders.Message) error {
			return microsoftSend(ctx, paymentsecrets.Actor{UserID: owner}, owner, microsoftmail.Message{To: []string{m.To}, Subject: m.Subject, HTML: m.HTML})
		})
	}
	if smtpSend != nil {
		senders[reminders.DeliveryModeSMTP] = reminders.SenderFunc(func(ctx context.Context, owner uuid.UUID, m reminders.Message) error {
			return smtpSend(ctx, paymentsecrets.Actor{UserID: owner}, owner, m.To, m.Subject, m.HTML)
		})
	}
	return senders
}
