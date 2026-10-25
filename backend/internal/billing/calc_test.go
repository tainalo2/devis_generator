package billing

import "testing"

func TestLineHTSimple(t *testing.T) {
	// 2 × 1000 cts = 2000
	ht, err := lineHT(2000, 1000, 0)
	if err != nil || ht != 2000 {
		t.Fatalf("ht=%d err=%v", ht, err)
	}
}

func TestLineHTFractionalQty(t *testing.T) {
	// 1.5 × 100 = 150
	ht, err := lineHT(1500, 100, 0)
	if err != nil || ht != 150 {
		t.Fatalf("ht=%d err=%v", ht, err)
	}
}

func TestLineHTDiscountAndHalfUp(t *testing.T) {
	// 1 × 101 cts, 10% discount → 90.9 → 91
	ht, err := lineHT(1000, 101, 1000)
	if err != nil || ht != 91 {
		t.Fatalf("ht=%d err=%v want 91", ht, err)
	}
}

func TestRoundHalfUpNegative(t *testing.T) {
	if got := roundHalfUp(-15, 10); got != -2 {
		t.Fatalf("got %d want -2", got)
	}
}

func TestComputeStandardMultiRate(t *testing.T) {
	res, err := Compute(RegimeStandard, []LineInput{
		{Description: "A", QuantityMilli: 1000, UnitPriceCents: 10000, TaxRateBps: 2000}, // HT 100, TVA 20
		{Description: "B", QuantityMilli: 1000, UnitPriceCents: 10000, TaxRateBps: 550},  // HT 100, TVA 5.5 → 6
		{Description: "C", QuantityMilli: 1000, UnitPriceCents: 333, TaxRateBps: 2000},   // HT 333
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.SubtotalCents != 10000+10000+333 {
		t.Fatalf("HT=%d", res.SubtotalCents)
	}
	// bucket 20%: (10000+333)*0.2 = 2066.6 → 2067
	// bucket 5.5%: 10000*0.055 = 550
	if res.TaxCents != 2067+550 {
		t.Fatalf("TVA=%d want %d", res.TaxCents, 2067+550)
	}
	if res.TotalCents != res.SubtotalCents+res.TaxCents {
		t.Fatalf("TTC inconsistent")
	}
}

func TestComputeFranchiseForcesZeroVAT(t *testing.T) {
	res, err := Compute(RegimeFranchise293B, []LineInput{
		{Description: "A", QuantityMilli: 1000, UnitPriceCents: 5000, TaxRateBps: 2000},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Lines[0].TaxRateBps != 0 || res.TaxCents != 0 || res.TotalCents != 5000 {
		t.Fatalf("franchise result: %+v", res)
	}
}

func TestComputeRejectsUnknownRate(t *testing.T) {
	_, err := Compute(RegimeStandard, []LineInput{
		{QuantityMilli: 1000, UnitPriceCents: 100, TaxRateBps: 999},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreditNoteNegativeUnitPrice(t *testing.T) {
	res, err := Compute(RegimeStandard, []LineInput{
		{Description: "Avoir", QuantityMilli: 1000, UnitPriceCents: -10000, TaxRateBps: 2000},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.SubtotalCents != -10000 || res.TaxCents != -2000 || res.TotalCents != -12000 {
		t.Fatalf("got HT=%d TVA=%d TTC=%d", res.SubtotalCents, res.TaxCents, res.TotalCents)
	}
}

func TestFormatNumber(t *testing.T) {
	if got := FormatNumber("F", 2026, 1); got != "F-2026-0001" {
		t.Fatalf("got %s", got)
	}
	p, err := PrefixForKind("deposit_invoice")
	if err != nil || p != "FA" {
		t.Fatalf("prefix=%s err=%v", p, err)
	}
}
