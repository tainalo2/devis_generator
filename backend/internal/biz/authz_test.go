package biz

import (
	"testing"

	"github.com/tainalo2/devis_generator/backend/internal/billing"
)

func TestIsImmutable(t *testing.T) {
	if !isImmutable("issued") || !isImmutable("paid") {
		t.Fatal("expected immutable")
	}
	if isImmutable("draft") {
		t.Fatal("draft should be editable")
	}
}

func TestValidateDocInput(t *testing.T) {
	err := validateDocInput(DocumentInput{Kind: "invoice", VATRegime: billing.RegimeStandard}, false)
	if err != nil {
		t.Fatal(err)
	}
	err = validateDocInput(DocumentInput{Kind: "nope", VATRegime: billing.RegimeStandard}, false)
	if err == nil {
		t.Fatal("expected invalid kind")
	}
}

func TestNumberingHelpers(t *testing.T) {
	p, err := billing.PrefixForKind("invoice")
	if err != nil || p != "F" {
		t.Fatalf("%s %v", p, err)
	}
	if billing.FormatNumber("FA", 2026, 12) != "FA-2026-0012" {
		t.Fatal("format")
	}
}
