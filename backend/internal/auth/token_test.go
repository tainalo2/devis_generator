package auth

import "testing"

func TestNewTokenDistinct(t *testing.T) {
	a, ha, err := NewToken(32)
	if err != nil {
		t.Fatal(err)
	}
	b, hb, err := NewToken(32)
	if err != nil {
		t.Fatal(err)
	}
	if a == b || ha == hb {
		t.Fatal("tokens should differ")
	}
	if HashToken(a) != ha {
		t.Fatal("hash mismatch")
	}
}
