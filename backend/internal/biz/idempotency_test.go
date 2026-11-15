package biz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func hashOf(t *testing.T, in DocumentInput) string {
	t.Helper()
	i, err := NewDocumentIdempotency(OpCreateDocument, "k", in)
	if err != nil {
		t.Fatal(err)
	}
	return i.RequestHash
}

func baseInput() DocumentInput {
	return DocumentInput{
		Kind: "quote", Title: "Devis", VATRegime: "standard", Currency: "EUR", Notes: "n", PaymentTerms: "30j",
		Lines: []DocLine{
			{Description: "A", QuantityMilli: 1000, UnitPriceCents: 10000, TaxRateBps: 2000},
			{Description: "B", QuantityMilli: 2500, UnitPriceCents: 500, DiscountBps: 1000, TaxRateBps: 2000},
		},
	}
}

func TestRequestHashAppliesCurrencyDefault(t *testing.T) {
	omitted, eur := baseInput(), baseInput()
	omitted.Currency = ""
	blank := baseInput()
	blank.Currency = "  "
	if hashOf(t, omitted) != hashOf(t, eur) || hashOf(t, blank) != hashOf(t, eur) {
		t.Fatal("devise omise/vide et EUR produisent le même document : le hash doit être identique")
	}
	usd := baseInput()
	usd.Currency = "USD"
	if hashOf(t, usd) == hashOf(t, eur) {
		t.Fatal("une autre devise est une différence métier réelle")
	}
}

func TestRequestHashDetectsRealBusinessDifferences(t *testing.T) {
	ref := hashOf(t, baseInput())
	mutations := map[string]func(*DocumentInput){
		"quantité":      func(in *DocumentInput) { in.Lines[0].QuantityMilli = 2000 },
		"prix":          func(in *DocumentInput) { in.Lines[0].UnitPriceCents = 10001 },
		"description":   func(in *DocumentInput) { in.Lines[0].Description = "A2" },
		"remise":        func(in *DocumentInput) { in.Lines[1].DiscountBps = 0 },
		"taux TVA":      func(in *DocumentInput) { in.Lines[0].TaxRateBps = 1000 },
		"ordre lignes":  func(in *DocumentInput) { in.Lines[0], in.Lines[1] = in.Lines[1], in.Lines[0] },
		"ligne en plus": func(in *DocumentInput) { in.Lines = append(in.Lines, in.Lines[0]) },
		"titre":         func(in *DocumentInput) { in.Title = "Autre" },
		"type":          func(in *DocumentInput) { in.Kind = "invoice" },
		"régime TVA":    func(in *DocumentInput) { in.VATRegime = "franchise_293b" },
		"notes":         func(in *DocumentInput) { in.Notes = "x" },
		"conditions":    func(in *DocumentInput) { in.PaymentTerms = "x" },
		"date":          func(in *DocumentInput) { d := "2026-01-01"; in.IssueDate = &d },
		"client":        func(in *DocumentInput) { id := uuid.New(); in.ClientID = &id },
	}
	for name, mutate := range mutations {
		in := baseInput()
		mutate(&in)
		if hashOf(t, in) == ref {
			t.Errorf("%s: le hash doit changer", name)
		}
	}
}

func TestRequestHashIgnoresDerivedFieldsAndKeyOrder(t *testing.T) {
	ref := hashOf(t, baseInput())
	withDerived := baseInput()
	for i := range withDerived.Lines {
		withDerived.Lines[i].ID = uuid.New()
		withDerived.Lines[i].Position = 7 + i
		withDerived.Lines[i].TaxCents = 999
		withDerived.Lines[i].TotalCents = 12345
	}
	if hashOf(t, withDerived) != ref {
		t.Fatal("id, position, tax_cents et total_cents n'ont aucun effet : ils ne doivent pas être hachés")
	}
	// Ordre des clés JSON et champs dérivés côté corps de requête.
	a := `{"kind":"quote","title":"Devis","vat_regime":"standard","currency":"EUR","notes":"n","payment_terms":"30j","lines":[` +
		`{"description":"A","quantity_milli":1000,"unit_price_cents":10000,"tax_rate_bps":2000},` +
		`{"description":"B","quantity_milli":2500,"unit_price_cents":500,"discount_bps":1000,"tax_rate_bps":2000}]}`
	b := `{"lines":[{"total_cents":5,"tax_rate_bps":2000,"unit_price_cents":10000,"position":3,"quantity_milli":1000,"description":"A","tax_cents":1},` +
		`{"tax_rate_bps":2000,"discount_bps":1000,"unit_price_cents":500,"quantity_milli":2500,"description":"B","id":"` + uuid.NewString() + `"}],` +
		`"payment_terms":"30j","notes":"n","vat_regime":"standard","title":"Devis","kind":"quote"}`
	var ina, inb DocumentInput
	if err := json.Unmarshal([]byte(a), &ina); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(b), &inb); err != nil {
		t.Fatal(err)
	}
	if hashOf(t, ina) != hashOf(t, inb) || hashOf(t, ina) != ref {
		t.Fatal("ordre des clés/champs dérivés : même requête métier, même hash attendu")
	}
	if len(ref) != 64 || strings.Trim(ref, "0123456789abcdef") != "" {
		t.Fatalf("hash=%q, want SHA-256 hex", ref)
	}
}

func TestLinkedAndConvertHashIgnoreDerivedFields(t *testing.T) {
	origin := uuid.New()
	lines := []DocLine{{Description: "Acompte", QuantityMilli: 1000, UnitPriceCents: 3000, TaxRateBps: 2000}}
	derived := []DocLine{{ID: uuid.New(), Position: 4, Description: "Acompte", QuantityMilli: 1000, UnitPriceCents: 3000, TaxRateBps: 2000, TaxCents: 600, TotalCents: 3000}}
	h := func(l []DocLine, o uuid.UUID) string {
		i, err := NewLinkedIdempotency(OpCreateDeposit, "k", o, l)
		if err != nil {
			t.Fatal(err)
		}
		return i.RequestHash
	}
	if h(lines, origin) != h(derived, origin) {
		t.Fatal("champs dérivés hachés pour acompte")
	}
	changed := []DocLine{{Description: "Acompte", QuantityMilli: 1000, UnitPriceCents: 3001, TaxRateBps: 2000}}
	if h(lines, origin) == h(changed, origin) || h(lines, origin) == h(lines, uuid.New()) {
		t.Fatal("prix/origine différents : hash différent attendu")
	}
	if h(nil, origin) != h([]DocLine{}, origin) {
		t.Fatal("lignes nil et vides sont équivalentes (copie de l'origine)")
	}
	c1, _ := NewConvertIdempotency("k", origin)
	c2, _ := NewConvertIdempotency("k", origin)
	c3, _ := NewConvertIdempotency("k", uuid.New())
	if c1.RequestHash != c2.RequestHash || c1.RequestHash == c3.RequestHash {
		t.Fatal("hash de conversion")
	}
}

type fakePurger struct {
	calls chan struct{}
	n     int64
	err   error
}

func (f *fakePurger) PurgeExpiredIdempotency(ctx context.Context) (int64, error) {
	f.calls <- struct{}{}
	return f.n, f.err
}

func TestIdempotencyPurgeLoopRunsOnStartAndEachTickAndStopsOnContext(t *testing.T) {
	p := &fakePurger{calls: make(chan struct{}, 10), n: 3}
	ticks := make(chan time.Time)
	ctx, cancel := context.WithCancel(context.Background())
	var logs bytes.Buffer
	done := make(chan struct{})
	go func() {
		RunIdempotencyPurgeLoop(ctx, p, ticks, slog.New(slog.NewTextHandler(&logs, nil)))
		close(done)
	}()
	wait := func(what string) {
		t.Helper()
		select {
		case <-p.calls:
		case <-time.After(2 * time.Second):
			t.Fatalf("pas de purge: %s", what)
		}
	}
	wait("au démarrage")
	ticks <- time.Now()
	wait("après un tick")
	p.err = errors.New("db down")
	ticks <- time.Now()
	wait("après un tick en erreur : la boucle continue")
	ticks <- time.Now()
	wait("après une erreur")
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("la boucle doit s'arrêter avec le contexte")
	}
	out := logs.String()
	if !strings.Contains(out, "purged=3") || !strings.Contains(out, "db down") {
		t.Fatalf("journalisation du nombre de lignes purgées et des erreurs manquante: %s", out)
	}
}

func TestRequestHashCanonicalizesTaxRateByRegime(t *testing.T) {
	withRate := func(regime string, rate int) DocumentInput {
		in := baseInput()
		in.VATRegime = regime
		for i := range in.Lines {
			in.Lines[i].TaxRateBps = rate
		}
		return in
	}
	// Franchise : le taux est forcé à 0 par billing.Compute ; omis == 0 == 2000.
	f0, f2000, f210 := hashOf(t, withRate("franchise_293b", 0)), hashOf(t, withRate("franchise_293b", 2000)), hashOf(t, withRate("franchise_293b", 210))
	if f0 != f2000 || f0 != f210 {
		t.Fatal("franchise_293b : taux omis, 0 et 2000 produisent le même document, le hash doit être identique")
	}
	// Hors franchise, le taux change le document.
	if hashOf(t, withRate("standard", 2000)) == hashOf(t, withRate("standard", 0)) {
		t.Fatal("standard : 2000 et 0 sont deux documents différents")
	}
	// Le régime reste une différence métier.
	if f0 == hashOf(t, withRate("standard", 0)) {
		t.Fatal("le régime doit rester dans le hash")
	}
}
