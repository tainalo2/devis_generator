package crypto

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 32)
	box, err := NewBox(key)
	if err != nil {
		t.Fatal(err)
	}

	plain := []byte(`{"line1":"10 rue Exemple","city":"Paris","iban":"FR761234567890"}`)
	blob, err := box.Encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	if blob[0] != CurrentKeyID {
		t.Fatalf("key id = %d, want %d", blob[0], CurrentKeyID)
	}
	got, err := box.Decrypt(blob)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("got %q want %q", got, plain)
	}
}

func TestDecryptTampered(t *testing.T) {
	box, err := NewBox(bytes.Repeat([]byte{1}, 32))
	if err != nil {
		t.Fatal(err)
	}
	blob, err := box.Encrypt([]byte("iban"))
	if err != nil {
		t.Fatal(err)
	}
	blob[len(blob)-1] ^= 0xff
	if _, err := box.Decrypt(blob); err == nil {
		t.Fatal("expected error on tampered blob")
	}
}

func TestNewBoxFromBase64(t *testing.T) {
	raw := bytes.Repeat([]byte{9}, 32)
	enc := base64.StdEncoding.EncodeToString(raw)
	box, err := NewBoxFromBase64(enc)
	if err != nil {
		t.Fatal(err)
	}
	out, err := box.EncryptString("FR76")
	if err != nil {
		t.Fatal(err)
	}
	s, err := box.DecryptString(out)
	if err != nil || s != "FR76" {
		t.Fatalf("roundtrip string: %q %v", s, err)
	}
}

func TestInvalidKeyLength(t *testing.T) {
	if _, err := NewBox([]byte("short")); err != ErrInvalidKey {
		t.Fatalf("err = %v, want ErrInvalidKey", err)
	}
}
