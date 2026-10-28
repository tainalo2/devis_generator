// Package facturx prépare une facture Factur-X au profil EN 16931.
//
// Build produit le XML CII. Il n'embarque pas ce XML dans un PDF/A-3 :
// le fichier téléchargé aujourd'hui reste un PDF simple.
package facturx

import (
	"encoding/xml"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/tainalo2/devis_generator/backend/internal/billing"
	"github.com/tainalo2/devis_generator/backend/internal/biz"
	"github.com/tainalo2/devis_generator/backend/internal/legal"
)

const (
	// GuidelineEN16931 est l'identifiant de profil Factur-X EN 16931.
	GuidelineEN16931 = "urn:cen.eu:en16931:2017#compliant#urn:factur-x.eu:1p0:en16931"

	nsRSM = "urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	nsQDT = "urn:un:unece:uncefact:data:standard:QualifiedDataType:100"
	nsRAM = "urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	nsUDT = "urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100"

	typeInvoice = "380"
	typePrepay  = "386"
	typeCredit  = "381"
	unitOne     = "C62"
	paymentSEPA = "30"
	schemeSIREN = "0002"
	schemeSIRET = "0009"
	schemeVAT   = "VA"
	vatExempt   = "E"
	vatStandard = "S"
	vatZero     = "Z"
)

var (
	errQuote     = errors.New("le devis n'est pas une facture Factur-X")
	errKind      = errors.New("type de document non pris en charge pour Factur-X")
	errNumber    = errors.New("numéro de facture manquant")
	errDate      = errors.New("date d'émission manquante")
	errSeller    = errors.New("nom du vendeur manquant")
	errBuyer     = errors.New("nom de l'acheteur manquant")
	errNoLines   = errors.New("au moins une ligne est requise")
	errTotals    = errors.New("totaux incohérents avec les lignes")
	errSellerVAT = errors.New("n° de TVA du vendeur manquant pour une facture au taux normal")
)

// Build écrit le XML CII EN 16931 d'une facture, d'une facture d'acompte ou d'un avoir.
func Build(doc biz.Document, issuer biz.IssuerProfile, client biz.Client) ([]byte, error) {
	code, err := documentType(doc.Kind)
	if err != nil {
		return nil, err
	}
	number := ""
	if doc.Number != nil {
		number = strings.TrimSpace(*doc.Number)
	}
	if number == "" {
		return nil, errNumber
	}
	if doc.IssueDate == nil || strings.TrimSpace(*doc.IssueDate) == "" {
		return nil, errDate
	}
	issued, err := parseDate(*doc.IssueDate)
	if err != nil {
		return nil, err
	}
	seller := strings.TrimSpace(issuer.LegalName)
	if seller == "" {
		return nil, errSeller
	}
	buyer := strings.TrimSpace(client.Name)
	if buyer == "" {
		return nil, errBuyer
	}
	if len(doc.Lines) == 0 {
		return nil, errNoLines
	}
	for i, l := range doc.Lines {
		if strings.TrimSpace(l.Description) == "" {
			return nil, fmt.Errorf("ligne %d : désignation manquante", i+1)
		}
		if l.QuantityMilli == 0 {
			return nil, fmt.Errorf("ligne %d : quantité nulle", i+1)
		}
	}

	currency := strings.ToUpper(strings.TrimSpace(doc.Currency))
	if currency == "" {
		currency = "EUR"
	}
	if len(currency) != 3 {
		return nil, fmt.Errorf("devise invalide")
	}

	res, err := billing.Compute(doc.VATRegime, lineInputs(doc.Lines))
	if err != nil {
		return nil, err
	}
	if res.SubtotalCents != doc.SubtotalCents || res.TaxCents != doc.TaxCents || res.TotalCents != doc.TotalCents {
		return nil, errTotals
	}
	for i, l := range res.Lines {
		if doc.Lines[i].TotalCents != l.TotalHTCents {
			return nil, fmt.Errorf("ligne %d : total HT incohérent", i+1)
		}
	}

	sellerCountry, err := countryCode(issuer.Address.Country)
	if err != nil {
		return nil, err
	}
	buyerCountry, err := countryCode(client.Address.Country)
	if err != nil {
		return nil, err
	}

	needsVAT := false
	for _, b := range res.Buckets {
		if vatCategory(doc.VATRegime, b.TaxRateBps) == vatStandard {
			needsVAT = true
			break
		}
	}
	vatID := strings.ToUpper(compactIBAN(issuer.VATNumber))
	if needsVAT && vatID == "" {
		return nil, errSellerVAT
	}

	inv := invoice{
		XmlnsRSM: nsRSM,
		XmlnsQDT: nsQDT,
		XmlnsRAM: nsRAM,
		XmlnsUDT: nsUDT,
		Context:  docContext{Guideline: idBlock{ID: GuidelineEN16931}},
		Document: exchanged{
			ID:       number,
			TypeCode: code,
			Issue:    date102(issued),
			Notes:    notesOf(doc),
		},
		Trade: transaction{
			Lines: make([]lineItem, 0, len(doc.Lines)),
			Agreement: agreement{
				Seller: partyOf(seller, issuer.TradeName, issuer.SIREN, issuer.SIRET, vatID, issuer.Email, issuer.Phone, issuer.Address, sellerCountry),
				Buyer:  partyOf(buyer, "", client.SIREN, client.SIRET, strings.ToUpper(compactIBAN(client.VATNumber)), client.Email, "", client.Address, buyerCountry),
			},
			Delivery: delivery{Event: &deliveryEvent{Occurrence: date102(issued)}},
			Settlement: settlement{
				Currency: currency,
				Sum: summation{
					LineTotal:  formatCents(res.SubtotalCents),
					TaxBasis:   formatCents(res.SubtotalCents),
					TaxTotal:   currencyAmount{Currency: currency, Value: formatCents(res.TaxCents)},
					GrandTotal: formatCents(res.TotalCents),
					DuePayable: formatCents(res.TotalCents),
				},
			},
		},
	}

	exemptReason := ""
	if doc.VATRegime == billing.RegimeFranchise293B {
		exemptReason = legal.DefaultMentions().Franchise293B
	}
	for _, b := range res.Buckets {
		cat := vatCategory(doc.VATRegime, b.TaxRateBps)
		tax := tradeTax{
			CalculatedAmount:      formatCents(b.TaxCents),
			TypeCode:              "VAT",
			BasisAmount:           formatCents(b.HTCents),
			CategoryCode:          cat,
			RateApplicablePercent: formatPercent(b.TaxRateBps),
		}
		if cat == vatExempt {
			tax.ExemptionReason = exemptReason
		}
		inv.Trade.Settlement.Taxes = append(inv.Trade.Settlement.Taxes, tax)
	}

	if iban := compactIBAN(issuer.IBAN); iban != "" {
		inv.Trade.Settlement.PaymentMeans = &paymentMeans{
			TypeCode: paymentSEPA,
			Account:  &account{IBAN: iban},
		}
	}
	if terms := paymentTerms(doc); terms != nil {
		inv.Trade.Settlement.PaymentTerms = terms
	}

	for i, src := range doc.Lines {
		calc := res.Lines[i]
		inv.Trade.Lines = append(inv.Trade.Lines, lineOf(i+1, src, calc, doc.VATRegime))
	}

	body, err := xml.MarshalIndent(inv, "", "  ")
	if err != nil {
		return nil, err
	}
	out := append([]byte(xml.Header), body...)
	out = append(out, '\n')
	return out, nil
}

func documentType(kind string) (string, error) {
	switch kind {
	case "quote":
		return "", errQuote
	case "invoice":
		return typeInvoice, nil
	case "deposit_invoice":
		return typePrepay, nil
	case "credit_note":
		return typeCredit, nil
	default:
		return "", errKind
	}
}

func lineInputs(lines []biz.DocLine) []billing.LineInput {
	out := make([]billing.LineInput, len(lines))
	for i, l := range lines {
		out[i] = billing.LineInput{
			Description:    l.Description,
			QuantityMilli:  l.QuantityMilli,
			UnitPriceCents: l.UnitPriceCents,
			DiscountBps:    l.DiscountBps,
			TaxRateBps:     l.TaxRateBps,
		}
	}
	return out
}

func vatCategory(regime string, bps int) string {
	if regime == billing.RegimeFranchise293B {
		return vatExempt
	}
	if bps == 0 {
		return vatZero
	}
	return vatStandard
}

func notesOf(doc biz.Document) []note {
	var out []note
	if title := strings.TrimSpace(doc.Title); title != "" {
		out = append(out, note{Content: "Objet : " + title})
	}
	if notes := strings.TrimSpace(doc.Notes); notes != "" {
		out = append(out, note{Content: notes})
	}
	if extra := strings.TrimSpace(doc.LegalMentions); extra != "" && extra != legal.DefaultMentions().Franchise293B {
		out = append(out, note{Content: extra})
	}
	return out
}

func partyOf(name, trade, siren, siret, vat, email, phone string, addr biz.Address, country string) party {
	p := party{
		Name: name,
		Address: address{
			Postcode: strings.TrimSpace(addr.PostalCode),
			LineOne:  strings.TrimSpace(addr.Line1),
			LineTwo:  strings.TrimSpace(addr.Line2),
			City:     strings.TrimSpace(addr.City),
			Country:  country,
		},
	}
	if id := siretOf(siret); id != "" {
		p.GlobalID = &schemeID{Scheme: schemeSIRET, Value: id}
	}
	if id := sirenOf(siren, siret); id != "" {
		org := &legalOrg{ID: schemeID{Scheme: schemeSIREN, Value: id}}
		if t := strings.TrimSpace(trade); t != "" && t != name {
			org.Trading = t
		}
		p.Legal = org
	}
	if vat != "" {
		p.TaxReg = &taxReg{ID: schemeID{Scheme: schemeVAT, Value: vat}}
	}
	email = strings.TrimSpace(email)
	phone = strings.TrimSpace(phone)
	if email != "" || phone != "" {
		c := &contact{PersonName: name}
		if phone != "" {
			c.Phone = &number{CompleteNumber: phone}
		}
		if email != "" {
			c.Email = &uri{URIID: email}
		}
		p.Contact = c
	}
	return p
}

func paymentTerms(doc biz.Document) *payTerms {
	desc := strings.TrimSpace(doc.PaymentTerms)
	var due *dateTime
	if doc.DueDate != nil && strings.TrimSpace(*doc.DueDate) != "" {
		if t, err := parseDate(*doc.DueDate); err == nil {
			d := date102(t)
			due = &d
		}
	}
	if desc == "" && due == nil {
		return nil
	}
	return &payTerms{Description: desc, Due: due}
}

func lineOf(n int, src biz.DocLine, calc billing.LineResult, regime string) lineItem {
	cat := vatCategory(regime, calc.TaxRateBps)
	gross := formatCents(src.UnitPriceCents)
	net := gross
	var allow *allowance
	if src.DiscountBps > 0 && src.QuantityMilli != 0 {
		net = formatUnitEuros(calc.TotalHTCents, src.QuantityMilli)
		// Remise unitaire = prix brut × taux de remise.
		perUnit := new(big.Rat).SetFrac(
			big.NewInt(src.UnitPriceCents*int64(src.DiscountBps)),
			big.NewInt(1_000_000),
		)
		allow = &allowance{
			ChargeIndicator: indicator{Value: false},
			ActualAmount:    formatRat(perUnit),
			Reason:          "Remise",
		}
	}
	return lineItem{
		Doc:     lineDoc{LineID: fmt.Sprintf("%d", n)},
		Product: product{Name: strings.TrimSpace(src.Description)},
		Agree: lineAgree{
			Gross: price{ChargeAmount: gross, Allowance: allow},
			Net:   price{ChargeAmount: net},
		},
		Delivery: lineDelivery{Billed: quantity{UnitCode: unitOne, Value: formatQty(src.QuantityMilli)}},
		Settle: lineSettle{
			Tax: tradeTax{
				TypeCode:              "VAT",
				CategoryCode:          cat,
				RateApplicablePercent: formatPercent(calc.TaxRateBps),
			},
			LineTotal: lineSum{LineTotalAmount: formatCents(calc.TotalHTCents)},
		},
	}
}

func date102(t time.Time) dateTime {
	return dateTime{Value: dateTimeString{Format: "102", Value: t.Format("20060102")}}
}
