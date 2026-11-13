package smtpconfig

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

type fakeStore struct {
	config  Config
	saves   int
	deletes int
	getErr  error
	saveErr error
}

func (s *fakeStore) Get(_ context.Context, _ uuid.UUID) (Config, error) {
	if s.getErr != nil {
		return Config{}, s.getErr
	}
	return s.config, nil
}
func (s *fakeStore) Save(_ context.Context, _ paymentsecrets.Actor, config Config, _ paymentsecrets.SealedSecret) (Config, error) {
	s.saves++
	if s.saveErr != nil {
		return Config{}, s.saveErr
	}
	s.config = config
	return config, nil
}
func (s *fakeStore) Delete(_ context.Context, _ uuid.UUID) error {
	s.deletes++
	return nil
}

type fakeVault struct {
	seals     int
	deletes   int
	plaintext []byte
}

func (v *fakeVault) Reveal(context.Context, paymentsecrets.Actor, uuid.UUID, paymentsecrets.Kind, string) ([]byte, paymentsecrets.Metadata, error) {
	return append([]byte(nil), v.plaintext...), paymentsecrets.Metadata{}, nil
}

func (v *fakeVault) Seal(plaintext []byte) (paymentsecrets.SealedSecret, error) {
	v.seals++
	v.plaintext = append([]byte(nil), plaintext...)
	return paymentsecrets.SealedSecret{Ciphertext: []byte{1, 2, 3}, MaskedValue: paymentsecrets.Mask(plaintext)}, nil
}
func (v *fakeVault) Delete(_ context.Context, _ paymentsecrets.Actor, _ uuid.UUID, _ paymentsecrets.Kind, _ string) error {
	v.deletes++
	return nil
}

type fakeTester struct {
	calls     int
	config    Config
	password  string
	err       error
	recipient string
	subject   string
}

func (t *fakeTester) Send(_ context.Context, config Config, password, recipient, subject, html string) error {
	t.calls++
	t.config, t.password, t.recipient, t.subject = config, password, recipient, subject
	return t.err
}

func (t *fakeTester) Test(_ context.Context, config Config, password string) error {
	t.calls++
	t.config = config
	t.password = password
	return t.err
}

func validInput() Input {
	return Input{
		Host: "smtp.example.com", Port: 587, Username: "mailer@example.com",
		Password: "smtp-secret", FromAddress: "mailer@example.com", TLSMode: TLSModeSTARTTLS,
	}
}

func TestSaveTestsSMTPBeforePersistingAndKeepsPasswordOutOfConfig(t *testing.T) {
	store, vault, tester := &fakeStore{}, &fakeVault{}, &fakeTester{}
	service := NewService(store, vault, tester)
	ownerID := uuid.New()
	actor := paymentsecrets.Actor{UserID: ownerID, RequestID: "req-save"}

	got, err := service.Save(context.Background(), actor, ownerID, validInput())
	if err != nil {
		t.Fatal(err)
	}
	if tester.calls != 1 || tester.password != "smtp-secret" {
		t.Fatalf("tester calls=%d password=%q", tester.calls, tester.password)
	}
	if vault.seals != 1 || string(vault.plaintext) != "smtp-secret" || store.saves != 1 {
		t.Fatalf("vault seals=%d store saves=%d", vault.seals, store.saves)
	}
	if got.Host != "smtp.example.com" || got.TLSMode != TLSModeSTARTTLS || !got.PasswordConfigured {
		t.Fatalf("unexpected config: %#v", got)
	}
}

func TestSaveDoesNotPersistWhenSMTPTestFails(t *testing.T) {
	store, vault := &fakeStore{}, &fakeVault{}
	tester := &fakeTester{err: errors.New("connection refused")}
	service := NewService(store, vault, tester)

	_, err := service.Save(context.Background(), paymentsecrets.Actor{UserID: uuid.New()}, uuid.New(), validInput())
	if !errors.Is(err, ErrSMTPTestFailed) {
		t.Fatalf("err=%v", err)
	}
	if vault.seals != 0 || store.saves != 0 {
		t.Fatalf("persisted after test failure: vault=%d store=%d", vault.seals, store.saves)
	}
}

func TestSaveFailureLeavesPreviousConfigurationUntouched(t *testing.T) {
	previous := Config{Host: "old.example.com", Port: 465, Username: "old"}
	store := &fakeStore{config: previous, saveErr: errors.New("database failed")}
	service := NewService(store, &fakeVault{}, &fakeTester{})
	_, err := service.Save(context.Background(), paymentsecrets.Actor{UserID: uuid.New()}, uuid.New(), validInput())
	if err == nil {
		t.Fatal("expected save failure")
	}
	if store.config != previous {
		t.Fatalf("previous state changed: %#v", store.config)
	}
}

func TestSaveRejectsInvalidConfigurationBeforeTesting(t *testing.T) {
	cases := []Input{
		{Host: "", Port: 587, Username: "u", Password: "p", FromAddress: "from@example.com", TLSMode: TLSModeSTARTTLS},
		{Host: "bad host", Port: 587, Username: "u", Password: "p", FromAddress: "from@example.com", TLSMode: TLSModeSTARTTLS},
		{Host: "smtp.example.com", Port: 0, Username: "u", Password: "p", FromAddress: "from@example.com", TLSMode: TLSModeSTARTTLS},
		{Host: "smtp.example.com", Port: 65536, Username: "u", Password: "p", FromAddress: "from@example.com", TLSMode: TLSModeSTARTTLS},
		{Host: "smtp.example.com", Port: 587, Username: "u", Password: "p", FromAddress: "not-an-email", TLSMode: TLSModeSTARTTLS},
		{Host: "smtp.example.com", Port: 587, Username: "u", Password: "p", FromAddress: "from@example.com", TLSMode: "plain"},
	}
	for i, input := range cases {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			tester := &fakeTester{}
			service := NewService(&fakeStore{}, &fakeVault{}, tester)
			if _, err := service.Save(context.Background(), paymentsecrets.Actor{UserID: uuid.New()}, uuid.New(), input); !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("err=%v", err)
			}
			if tester.calls != 0 {
				t.Fatal("invalid config was tested")
			}
		})
	}
}

func TestDeleteRemovesSecretAndConfiguration(t *testing.T) {
	store, vault := &fakeStore{}, &fakeVault{}
	service := NewService(store, vault, &fakeTester{})
	ownerID := uuid.New()
	actor := paymentsecrets.Actor{UserID: ownerID, RequestID: "req-delete"}

	if err := service.Delete(context.Background(), actor, ownerID); err != nil {
		t.Fatal(err)
	}
	if vault.deletes != 1 || store.deletes != 1 {
		t.Fatalf("vault deletes=%d store deletes=%d", vault.deletes, store.deletes)
	}
}

func TestGetReturnsNonSecretConfig(t *testing.T) {
	store := &fakeStore{config: Config{OwnerID: uuid.New(), Host: "smtp.example.com", Port: 465, TLSMode: TLSModeTLS, PasswordConfigured: true}}
	service := NewService(store, &fakeVault{}, &fakeTester{})
	got, err := service.Get(context.Background(), store.config.OwnerID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Host != store.config.Host || !got.PasswordConfigured {
		t.Fatalf("unexpected config: %#v", got)
	}
}

func TestSendMailRevealsPasswordAndUsesStoredConfiguration(t *testing.T) {
	owner := uuid.New()
	store := &fakeStore{config: Config{OwnerID: owner, Host: "smtp.example.com", Port: 465, Username: "mailer", FromAddress: "mailer@example.com", TLSMode: TLSModeTLS}}
	vault := &fakeVault{plaintext: []byte("secret")}
	tester := &fakeTester{}
	service := NewService(store, vault, tester)
	if err := service.SendMail(context.Background(), paymentsecrets.Actor{UserID: owner, RequestID: "reminder"}, owner, "client@example.com", "Facture", "<p>Bonjour</p>"); err != nil {
		t.Fatal(err)
	}
	if tester.recipient != "client@example.com" || tester.subject != "Facture" || tester.password != "secret" {
		t.Fatalf("recipient=%q subject=%q password=%q", tester.recipient, tester.subject, tester.password)
	}
}
