package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	p := DefaultArgon2Params()
	p.Memory = 8 * 1024 // faster tests
	p.Time = 1

	hash, err := HashPassword("MotDePasseCorrect1!", p)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := VerifyPassword("MotDePasseCorrect1!", hash)
	if err != nil || !ok {
		t.Fatalf("verify ok=%v err=%v", ok, err)
	}
	ok, err = VerifyPassword("mauvais", hash)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected mismatch")
	}
}

func TestHashPasswordEmpty(t *testing.T) {
	if _, err := HashPassword("", DefaultArgon2Params()); err == nil {
		t.Fatal("expected error")
	}
}
