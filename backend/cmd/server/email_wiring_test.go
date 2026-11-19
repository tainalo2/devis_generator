package main

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/config"
	"github.com/tainalo2/devis_generator/backend/internal/gmailmail"
	"github.com/tainalo2/devis_generator/backend/internal/microsoftmail"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
	"github.com/tainalo2/devis_generator/backend/internal/reminders"
)

func TestBuildEmailServicesAlwaysWiresRemindersAndSMTP(t *testing.T) {
	svc, err := buildEmailServices(nil, nil, config.Config{SessionSecret: "0123456789abcdef0123456789abcdef"})
	if err != nil {
		t.Fatal(err)
	}
	if svc.Reminders == nil || svc.SMTP == nil {
		t.Fatalf("reminders=%v smtp=%v: ils doivent toujours être câblés", svc.Reminders, svc.SMTP)
	}
	if svc.Gmail != nil || svc.Microsoft != nil {
		t.Fatal("sans client OAuth, Gmail et Microsoft ne doivent pas être câblés")
	}
}

func TestBuildEmailServicesWiresOAuthProvidersWhenConfigured(t *testing.T) {
	cfg := config.Config{
		SessionSecret:       "0123456789abcdef0123456789abcdef",
		GoogleOAuthClientID: "g-id", GoogleOAuthClientSecret: "g-secret", GoogleOAuthRedirectURL: "https://app.example.test/api/v1/email/gmail/callback",
		MicrosoftOAuthClientID: "m-id", MicrosoftOAuthClientSecret: "m-secret", MicrosoftOAuthRedirectURL: "https://app.example.test/api/v1/email/microsoft/callback",
	}
	svc, err := buildEmailServices(nil, &paymentsecrets.Vault{}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if svc.Gmail == nil || svc.Microsoft == nil {
		t.Fatalf("gmail=%v microsoft=%v", svc.Gmail, svc.Microsoft)
	}
	url, err := svc.Gmail.AuthorizationURL(uuid.New())
	if err != nil || len(url) < 20 || url[:8] != "https://" {
		t.Fatalf("url=%q err=%v", url, err)
	}
}

func TestBuildEmailServicesRejectsEmptySessionSecret(t *testing.T) {
	cfg := config.Config{GoogleOAuthClientID: "g", GoogleOAuthClientSecret: "s", GoogleOAuthRedirectURL: "https://x/cb"}
	if _, err := buildEmailServices(nil, &paymentsecrets.Vault{}, cfg); err == nil {
		t.Fatal("un secret de session vide ne doit pas servir à dériver la clé d'état OAuth")
	}
}

func TestOAuthStateKeysAreDerivedAndSeparatedPerProvider(t *testing.T) {
	a, b := oauthStateKey("secret-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", "gmail"), oauthStateKey("secret-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", "microsoft")
	if len(a) != 32 || string(a) == string(b) {
		t.Fatalf("clés d'état: len=%d identiques=%v", len(a), string(a) == string(b))
	}
}

func TestReminderSendersRouteMessagesToTheOwnersProvider(t *testing.T) {
	var gotGmail gmailmail.Message
	var gotMS microsoftmail.Message
	var gotSMTP [3]string
	senders := reminderSenders(
		func(_ context.Context, a paymentsecrets.Actor, owner uuid.UUID, m gmailmail.Message) error {
			if a.UserID != owner {
				t.Fatal("l'acteur doit être le propriétaire")
			}
			gotGmail = m
			return nil
		},
		func(_ context.Context, _ paymentsecrets.Actor, _ uuid.UUID, m microsoftmail.Message) error {
			gotMS = m
			return nil
		},
		func(_ context.Context, _ paymentsecrets.Actor, _ uuid.UUID, to, subject, html string) error {
			gotSMTP = [3]string{to, subject, html}
			return errors.New("smtp down")
		},
	)
	msg := reminders.Message{To: "client@example.test", Subject: "Sujet", HTML: "<p>x</p>"}
	owner := uuid.New()
	if err := senders[reminders.DeliveryModeGmail].Send(context.Background(), owner, msg); err != nil || gotGmail.To[0] != "client@example.test" || gotGmail.Subject != "Sujet" || gotGmail.HTML != "<p>x</p>" {
		t.Fatalf("gmail: %v %+v", err, gotGmail)
	}
	if err := senders[reminders.DeliveryModeMicrosoft].Send(context.Background(), owner, msg); err != nil || gotMS.To[0] != "client@example.test" {
		t.Fatalf("microsoft: %v %+v", err, gotMS)
	}
	if err := senders[reminders.DeliveryModeSMTP].Send(context.Background(), owner, msg); err == nil || gotSMTP != [3]string{"client@example.test", "Sujet", "<p>x</p>"} {
		t.Fatalf("smtp: %v %v", err, gotSMTP)
	}
}

func TestReminderSendersSkipUnconfiguredProviders(t *testing.T) {
	senders := reminderSenders(nil, nil, nil)
	if len(senders) != 0 {
		t.Fatalf("aucun fournisseur câblé: %v", senders)
	}
}
