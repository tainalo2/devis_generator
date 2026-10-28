package biz

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/tainalo2/devis_generator/backend/internal/crypto"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	box, err := crypto.NewBox(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	return NewStore(nil, box)
}

func sampleIssuer() IssuerProfile {
	return IssuerProfile{
		LegalName: "Atelier Test",
		SIRET:     "12345678900012",
		IBAN:      "FR7630006000011234567890189",
		Address:   Address{Line1: "1 rue des Lilas", PostalCode: "75001", City: "Paris", Country: "FR"},
	}
}

func TestIssuerSnapshotSealedAtRest(t *testing.T) {
	s := testStore(t)
	raw, err := s.sealIssuerSnapshot(sampleIssuer())
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"FR7630006000011234567890189", "1 rue des Lilas", "Paris"} {
		if bytes.Contains(raw, []byte(secret)) {
			t.Fatalf("instantané stocké contient %q en clair : %s", secret, raw)
		}
	}
	if !bytes.Contains(raw, []byte("Atelier Test")) {
		t.Fatalf("les champs publics doivent rester lisibles : %s", raw)
	}
	if isPlainSnapshot(raw) {
		t.Fatal("un instantané chiffré ne doit pas être vu comme en clair")
	}

	got, err := s.openIssuerSnapshot(raw)
	if err != nil {
		t.Fatal(err)
	}
	want := sampleIssuer()
	if got.IBAN != want.IBAN || got.Address != want.Address || got.LegalName != want.LegalName {
		t.Fatalf("aller-retour incorrect : %+v", got)
	}
}

func TestIssuerSnapshotLegacyPlaintext(t *testing.T) {
	s := testStore(t)
	legacy, _ := json.Marshal(sampleIssuer())
	if !isPlainSnapshot(legacy) {
		t.Fatal("un ancien instantané en clair doit être détecté")
	}
	got, err := s.openIssuerSnapshot(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if got.IBAN != sampleIssuer().IBAN || got.Address.City != "Paris" {
		t.Fatalf("lecture d'un ancien instantané incorrecte : %+v", got)
	}
}

func TestClientSnapshotSealedAtRest(t *testing.T) {
	s := testStore(t)
	in := ClientSnapshot{
		Name:    "Studio Lumen",
		Email:   "compta@lumen.example",
		SIREN:   "123456789",
		Address: Address{Line1: "2 rue des Ateliers", PostalCode: "69001", City: "Lyon", Country: "FR"},
	}
	raw, err := s.sealClientSnapshot(in)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("2 rue des Ateliers")) || bytes.Contains(raw, []byte("Lyon")) {
		t.Fatalf("adresse du client en clair : %s", raw)
	}
	if !bytes.Contains(raw, []byte("Studio Lumen")) {
		t.Fatalf("le nom doit rester lisible (liste des documents) : %s", raw)
	}
	got, err := s.openClientSnapshot(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got != in {
		t.Fatalf("aller-retour incorrect : %+v", got)
	}
}
