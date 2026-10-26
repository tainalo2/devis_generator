package biz

import "testing"

func TestStatusTransitionAllowed(t *testing.T) {
	cases := []struct {
		kind, from, to string
		want           bool
	}{
		{"invoice", "issued", "sent", true},
		{"invoice", "issued", "paid", true},
		{"invoice", "sent", "paid", true},
		{"deposit_invoice", "sent", "paid", true},
		{"invoice", "draft", "paid", false},
		{"invoice", "paid", "sent", false},
		{"invoice", "issued", "accepted", false},
		{"quote", "issued", "accepted", true},
		{"quote", "sent", "rejected", true},
		{"quote", "issued", "paid", false},
		{"credit_note", "issued", "paid", false},
		{"credit_note", "issued", "sent", true},
		{"invoice", "issued", "cancelled", false},
	}
	for _, tc := range cases {
		if got := StatusTransitionAllowed(tc.kind, tc.from, tc.to); got != tc.want {
			t.Errorf("StatusTransitionAllowed(%q, %q, %q) = %v, want %v", tc.kind, tc.from, tc.to, got, tc.want)
		}
	}
}
