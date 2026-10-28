package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGuestPDF(t *testing.T) {
	s := New(Options{Version: "test"})
	const body = `{
		"kind": "invoice",
		"vat_regime": "franchise_293b",
		"title": "Prestation « été »",
		"issue_date": "2026-10-28",
		"payment_terms": "À réception",
		"notes": "Merci — à bientôt.",
		"issuer": {
			"legal_name": "Atelier Été",
			"siret": "12345678900012",
			"email": "ete@exemple.fr",
			"address": {"line1": "12 rue des Érables", "postal_code": "69001", "city": "Lyon", "country": "France"}
		},
		"client": {"name": "Société Cœur", "siren": "123456789"},
		"lines": [
			{"description": "Création d’un logo", "quantity_milli": 1000, "unit_price_cents": 10000, "tax_rate_bps": 0}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/guest/pdf", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Fatalf("content-type=%q", ct)
	}
	pdf := rec.Body.Bytes()
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Fatal("réponse sans en-tête PDF")
	}
	if !bytes.Contains(pdf, []byte("/Identity-H")) {
		t.Fatal("police Unicode absente")
	}
	if !bytes.Contains(pdf, []byte("/Subtype /Image")) && !bytes.Contains(pdf, []byte("/Subtype/Image")) {
		t.Fatal("logo absent")
	}
	if !bytes.Contains(pdf, []byte("/Image")) {
		t.Fatal("flux image absent")
	}
}
