package emailtemplates

import "testing"

func TestDefaultsRender(t *testing.T) {
	for _, k := range []Kind{QuoteUnsigned, InvoiceUnpaid} {
		tt := Template{Kind: k, Subject: defaultSubject(k), Body: defaultBody(k)}
		s, b, err := Render(tt, Vars{ClientName: "Alice", DocRef: "D-1", Amount: "100 EUR", Link: "https://x/d/tok", DueDate: "2026-12-01"})
		if err != nil {
			t.Fatal(err)
		}
		if s == "" || b == "" {
			t.Fatal("empty render")
		}
	}
}
