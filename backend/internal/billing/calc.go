// Package billing computes HT / TVA / TTC totals in integer cents.
//
// Rounding rules (documented + tested):
//  1. Line HT = round_half_up(quantity × unit_price × (1 − discount))
//     quantity has 3 decimal places (stored as milli-units: 1000 = 1.000).
//  2. Lines are grouped by tax rate (basis points).
//  3. For each rate: TVA = round_half_up(sum(HT_line) × rate / 10_000).
//  4. Totals: HT = Σ line HT, TVA = Σ bucket TVA, TTC = HT + TVA.
//  5. franchise_293b: all rates forced to 0, TVA = 0, TTC = HT.
//  6. Credit notes use the same formulas; the caller may negate for display.
package billing

import (
	"errors"
	"fmt"
	"sort"
)

// Allowed VAT rates in basis points (0.00 %, 2.10 %, 5.50 %, 10 %, 20 %).
var AllowedTaxRatesBPS = []int{0, 210, 550, 1000, 2000}

const (
	RegimeFranchise293B = "franchise_293b"
	RegimeStandard      = "standard"
)

// LineInput is one document line before totals.
type LineInput struct {
	Description    string
	QuantityMilli  int64 // 1000 = 1.000
	UnitPriceCents int64 // can be negative for explicit credit lines
	DiscountBps    int   // 0..10000
	TaxRateBps     int
}

// LineResult is a priced line.
type LineResult struct {
	Description    string `json:"description"`
	QuantityMilli  int64  `json:"quantity_milli"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	DiscountBps    int    `json:"discount_bps"`
	TaxRateBps     int    `json:"tax_rate_bps"`
	TotalHTCents   int64  `json:"total_ht_cents"`
}

// TaxBucket aggregates HT/TVA for one rate.
type TaxBucket struct {
	TaxRateBps int   `json:"tax_rate_bps"`
	HTCents    int64 `json:"ht_cents"`
	TaxCents   int64 `json:"tax_cents"`
}

// Result is the full document computation.
type Result struct {
	Lines         []LineResult `json:"lines"`
	Buckets       []TaxBucket  `json:"buckets"`
	SubtotalCents int64        `json:"subtotal_cents"` // HT
	TaxCents      int64        `json:"tax_cents"`
	TotalCents    int64        `json:"total_cents"` // TTC
	VatRegime     string       `json:"vat_regime"`
}

// Compute prices a document.
func Compute(regime string, lines []LineInput) (Result, error) {
	if regime != RegimeFranchise293B && regime != RegimeStandard {
		return Result{}, fmt.Errorf("régime TVA inconnu: %s", regime)
	}
	out := Result{VatRegime: regime, Lines: make([]LineResult, 0, len(lines))}
	bucketHT := map[int]int64{}

	for i, in := range lines {
		if in.QuantityMilli == 0 {
			return Result{}, fmt.Errorf("ligne %d: quantité nulle", i+1)
		}
		if in.DiscountBps < 0 || in.DiscountBps > 10000 {
			return Result{}, fmt.Errorf("ligne %d: remise invalide", i+1)
		}
		rate := in.TaxRateBps
		if regime == RegimeFranchise293B {
			rate = 0
		} else if !isAllowedRate(rate) {
			return Result{}, fmt.Errorf("ligne %d: taux TVA non autorisé (%d bps)", i+1, rate)
		}

		ht, err := lineHT(in.QuantityMilli, in.UnitPriceCents, in.DiscountBps)
		if err != nil {
			return Result{}, fmt.Errorf("ligne %d: %w", i+1, err)
		}
		out.Lines = append(out.Lines, LineResult{
			Description:    in.Description,
			QuantityMilli:  in.QuantityMilli,
			UnitPriceCents: in.UnitPriceCents,
			DiscountBps:    in.DiscountBps,
			TaxRateBps:     rate,
			TotalHTCents:   ht,
		})
		bucketHT[rate] += ht
		out.SubtotalCents += ht
	}

	rates := make([]int, 0, len(bucketHT))
	for r := range bucketHT {
		rates = append(rates, r)
	}
	sort.Ints(rates)
	for _, r := range rates {
		ht := bucketHT[r]
		tax := int64(0)
		if regime == RegimeStandard && r > 0 {
			tax = roundHalfUp(ht*int64(r), 10000)
		}
		out.Buckets = append(out.Buckets, TaxBucket{TaxRateBps: r, HTCents: ht, TaxCents: tax})
		out.TaxCents += tax
	}
	out.TotalCents = out.SubtotalCents + out.TaxCents
	return out, nil
}

func lineHT(qtyMilli, unitCents int64, discountBps int) (int64, error) {
	// ht = qtyMilli * unitCents * (10000 - discount) / (1000 * 10000)
	numer := qtyMilli * unitCents * int64(10000-discountBps)
	denom := int64(1000 * 10000)
	if denom == 0 {
		return 0, errors.New("dénominateur nul")
	}
	return roundHalfUp(numer, denom), nil
}

// roundHalfUp divides numer/denom with half-away-from-zero / commercial rounding.
func roundHalfUp(numer, denom int64) int64 {
	if denom < 0 {
		numer = -numer
		denom = -denom
	}
	if numer >= 0 {
		return (numer + denom/2) / denom
	}
	// negative: half away from zero
	return -((-numer + denom/2) / denom)
}

func isAllowedRate(bps int) bool {
	for _, a := range AllowedTaxRatesBPS {
		if a == bps {
			return true
		}
	}
	return false
}

// PrefixForKind returns the numbering prefix for a document kind.
func PrefixForKind(kind string) (string, error) {
	switch kind {
	case "quote":
		return "D", nil
	case "invoice":
		return "F", nil
	case "deposit_invoice":
		return "FA", nil
	case "credit_note":
		return "A", nil
	default:
		return "", fmt.Errorf("type de document inconnu: %s", kind)
	}
}

// FormatNumber builds e.g. F-2026-0001.
func FormatNumber(prefix string, year, seq int) string {
	return fmt.Sprintf("%s-%d-%04d", prefix, year, seq)
}
