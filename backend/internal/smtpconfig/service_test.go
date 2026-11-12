package smtpconfig

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/tainalo2/devis_generator/backend/internal/paymentsecrets"
)

type fakeStore struct {
	config    Config
	upserts   int
	deletes   int
	getErr    error
	upsertErr error
}

func (s *fakeStore) Get(_ context.Context, _ uuid.UUID) (Config, error) {
	if s.getErr != nil {
		return Config{}, s.getErr
	}
	return s.config, nil
}
func (s *fakeStore) Upsert(_ context.Context, config Config) (Config, error) {
	s.upserts++
	if s.upsertErr != nil {
		return Config{}, s.upsertErr
	}
	s.config = config
	return config, nil
}
func (s *fakeStore) Delete(_ context.Context, _ uuid.UUID) error {
	s.deletes++
	return nil
}

type fakeVault struct {
	puts      int
	deletes   int
	plaintext []byte
}

func (v *fakeVault) Put(_ context.Context, _ paymentsecrets.Actor, ownerID uuid.UUID, kind paymentsecrets.Kind, reference string, plaintext []byte) (paymentsecrets.Metadata, error) {
	v.puts++
	v.plaintext = append([]byte(nil), plaintext...)
	return paymentsecrets.Metadata{OwnerID: ownerID, Kind: kind, Reference: reference}, nil
}
func (v *fakeVault) Delete(_ context.Context, _ paymentsecrets.Actor, _ uuid.UUID, _ paymentsecrets.Kind, _ string) error {
	v.deletes++
	return nil
}

type fakeTester struct {
	calls    int
	config   Config
	password string
	err      error
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
	if vault.puts != 1 || string(vault.plaintext) != "smtp-secret" || store.upserts != 1 {
		t.Fatalf("vault puts=%d store upserts=%d", vault.puts, store.upserts)
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
	if vault.puts != 0 || store.upserts != 0 {
		t.Fatalf("persisted after test failure: vault=%d store=%d", vault.puts, store.upserts)
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
