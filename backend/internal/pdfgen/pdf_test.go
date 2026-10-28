package pdfgen

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tainalo2/devis_generator/backend/internal/billing"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/legal"
)

func TestRenderPDFMinimal(t *testing.T) {
	num := "F-2026-0001"
	date := "2026-10-25"
	doc := biz.Document{
		Kind: "invoice", Status: "issued", Number: &num, IssueDate: &date,
		VATRegime: billing.RegimeFranchise293B, Currency: "EUR",
		SubtotalCents: 10000, TaxCents: 0, TotalCents: 10000,
		Lines: []biz.DocLine{{
			Description: "Prestation", QuantityMilli: 1000, UnitPriceCents: 10000, TotalCents: 10000,
		}},
		PaymentTerms: "À réception",
	}
	pdf, err := Render(doc, biz.IssuerProfile{LegalName: "AE Exemple", SIRET: "12345678900012"},
		biz.Client{Name: "Client SAS"}, legal.DefaultMentions())
	if err != nil {
		t.Fatal(err)
	}
	if len(pdf) < 100 || string(pdf[:4]) != "%PDF" {
		t.Fatalf("not a pdf: len=%d", len(pdf))
	}
	if !bytes.Contains(inflateStreams(pdf), utf16BE("293 B")) {
		t.Fatal("mention franchise absente")
	}
	if !bytes.Contains(inflateStreams(pdf), utf16BE("À réception")) {
		t.Fatal("accent du délai de paiement absent")
	}
}

func TestFormatMoneyQtyRate(t *testing.T) {
	if got := formatEUR(123456); got != "1"+nbsp+"234,56"+nbsp+"€" {
		t.Fatalf("eur=%q", got)
	}
	if got := formatEUR(-50); got != "\u2212"+"0,50"+nbsp+"€" {
		t.Fatalf("neg=%q", got)
	}
	if got := formatQty(1500); got != "1,5" {
		t.Fatalf("qty=%q", got)
	}
	if got := formatQty(1001); got != "1,001" {
		t.Fatalf("qty milli=%q", got)
	}
	if got := formatRate(550); got != "5,5 %" {
		t.Fatalf("rate=%q", got)
	}
	if got := formatDate("2026-10-28"); got != "28/10/2026" {
		t.Fatalf("date=%q", got)
	}
}

func TestRenderUTF8ImageAndLayout(t *testing.T) {
	num := "D-2026-0042"
	issue := "2026-10-28"
	due := "2026-11-28"
	doc := biz.Document{
		Kind: "quote", Status: "draft", Number: &num, IssueDate: &issue, DueDate: &due,
		Title:     "Création d’un site « été » — cœur & île",
		VATRegime: billing.RegimeStandard, Currency: "EUR",
		Notes:         "Merci.\nÀ bientôt — l’équipe.",
		PaymentTerms:  "À réception",
		SubtotalCents: 185000, TaxCents: 33665, TotalCents: 218665,
		Lines: []biz.DocLine{
			{Description: "Maquette « été », ½ journée", QuantityMilli: 1500, UnitPriceCents: 120000, TaxRateBps: 2000, DiscountBps: 1000, TotalCents: 162000},
			{Description: "Hébergement — année", QuantityMilli: 1000, UnitPriceCents: 23000, TaxRateBps: 550, TotalCents: 23000},
		},
		LegalMentions: "Pénalité d’escompte : néant.",
	}
	// Totaux volontairement distincts des lignes : le PDF affiche les montants stockés.
	pdf, err := Render(doc, biz.IssuerProfile{
		LegalName: "Atelier Été & Cœur", TradeName: "Atelier Été",
		Email: "été@exemple.fr", Phone: "06 12 34 56 78",
		SIRET: "12345678900012", VATNumber: "FR12123456789",
		APECode: "62.01Z", RCS: "Lyon",
		IBAN:    "FR76 3000 6000 0112 3456 7890 189",
		Address: biz.Address{Line1: "12 rue des Érables", PostalCode: "69001", City: "Lyon", Country: "France"},
	}, biz.Client{
		Name:    "Société Générale d'Île-de-France",
		Email:   "contact@ile.fr",
		Address: biz.Address{Line1: "1 place de l'Île", PostalCode: "75004", City: "Paris"},
		SIREN:   "123456789",
	}, legal.DefaultMentions())
	if err != nil {
		t.Fatal(err)
	}
	if string(pdf[:4]) != "%PDF" {
		t.Fatalf("not a pdf")
	}
	if !bytes.Contains(pdf, []byte("/Subtype /Image")) && !bytes.Contains(pdf, []byte("/Subtype/Image")) {
		t.Fatal("PDF sans image (logo)")
	}
	if !bytes.Contains(pdf, []byte("/Identity-H")) {
		t.Fatal("PDF sans police Unicode")
	}
	plain := inflateStreams(pdf)
	for _, phrase := range []string{
		"Atelier Été",
		"« été »",
		"Cœur",
		"Île-de-France",
		"1" + nbsp + "200,00" + nbsp + "€",
		"Bon pour accord",
		"Pénalité d’escompte",
		"5,5 %",
	} {
		if !bytes.Contains(plain, utf16BE(phrase)) {
			t.Errorf("texte absent du PDF: %q", phrase)
		}
	}
	// Couleur d’accent #2f9e94 ≈ 0.184 0.620 0.580.
	if !bytes.Contains(plain, []byte("0.184")) && !bytes.Contains(plain, []byte("0.620")) {
		t.Error("couleur d’accent absente")
	}

	if os.Getenv("PDF_SAMPLE") != "" {
		path := filepath.Join(os.TempDir(), "devis-sample.pdf")
		if err := os.WriteFile(path, pdf, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Log(path)
	}
}

func TestRenderMultiPage(t *testing.T) {
	num := "F-2026-0099"
	date := "2026-10-28"
	lines := make([]biz.DocLine, 0, 36)
	var sub int64
	for i := 1; i <= 36; i++ {
		lines = append(lines, biz.DocLine{
			Description:    fmt.Sprintf("Préparation n° %02d — fournitures spéciales pour l’atelier", i),
			QuantityMilli:  2000,
			UnitPriceCents: 1500,
			TaxRateBps:     0,
			TotalCents:     3000,
		})
		sub += 3000
	}
	doc := biz.Document{
		Kind: "invoice", Number: &num, IssueDate: &date,
		VATRegime:     billing.RegimeFranchise293B,
		Title:         "Chantier",
		SubtotalCents: sub, TotalCents: sub,
		Lines: lines,
	}
	pdf, err := Render(doc, biz.IssuerProfile{LegalName: "Émile", SIRET: "12345678900012"},
		biz.Client{Name: "Mairie d’Århus"}, legal.DefaultMentions())
	if err != nil {
		t.Fatal(err)
	}
	plain := inflateStreams(pdf)
	if !bytes.Contains(plain, utf16BE("suite")) {
		t.Fatal("pas de report de page")
	}
	if !bytes.Contains(plain, utf16BE("Préparation n° 36")) {
		t.Fatal("dernière ligne absente")
	}
	if !bytes.Contains(plain, utf16BE("Mairie d’Århus")) {
		t.Fatal("caractère Å absent")
	}
}

func TestRenderEmptyAndCredit(t *testing.T) {
	pdf, err := Render(biz.Document{Kind: "credit_note", VATRegime: billing.RegimeFranchise293B},
		biz.IssuerProfile{}, biz.Client{}, legal.DefaultMentions())
	if err != nil {
		t.Fatal(err)
	}
	plain := inflateStreams(pdf)
	if !bytes.Contains(plain, utf16BE("AVOIR")) || !bytes.Contains(plain, utf16BE("Montant de l’avoir")) {
		t.Fatal("avoir incomplet")
	}
	if string(pdf[:4]) != "%PDF" {
		t.Fatal("not a pdf")
	}
}

func utf16BE(s string) []byte {
	var b bytes.Buffer
	for _, r := range s {
		b.WriteByte(byte(r >> 8))
		b.WriteByte(byte(r))
	}
	return b.Bytes()
}

func inflateStreams(pdf []byte) []byte {
	var out bytes.Buffer
	out.Write(pdf)
	rest := pdf
	for {
		i := bytes.Index(rest, []byte("stream"))
		if i < 0 {
			break
		}
		rest = rest[i+len("stream"):]
		if len(rest) > 0 && rest[0] == '\r' {
			rest = rest[1:]
		}
		if len(rest) > 0 && rest[0] == '\n' {
			rest = rest[1:]
		}
		j := bytes.Index(rest, []byte("endstream"))
		if j < 0 {
			break
		}
		raw := rest[:j]
		raw = bytes.TrimRight(raw, "\r\n")
		if dec := inflate(raw); len(dec) > 0 {
			out.Write(dec)
		}
		rest = rest[j+len("endstream"):]
	}
	return out.Bytes()
}

func inflate(raw []byte) []byte {
	if zr, err := zlib.NewReader(bytes.NewReader(raw)); err == nil {
		b, err := io.ReadAll(zr)
		zr.Close()
		if err == nil {
			return b
		}
	}
	fr := flate.NewReader(bytes.NewReader(raw))
	defer fr.Close()
	b, err := io.ReadAll(fr)
	if err != nil {
		return nil
	}
	return b
}

func TestCleanStripsControls(t *testing.T) {
	got := clean("a\x00b\tété\U0001F600")
	if strings.Contains(got, "\x00") || !strings.Contains(got, "été") || !strings.Contains(got, "?") {
		t.Fatalf("%q", got)
	}
}
