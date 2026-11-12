package bankcoord

import "testing"

func TestValidateIBAN(t *testing.T) {
	cases := []struct {
		in   string
		ok   bool
	}{
		{"FR76 3000 6000 0112 3456 7890 189", true},
		{"DE89370400440532013000", true},
		{"FR00 3000 6000 0112 3456 7890 189", false},
		{"notaniban", false},
		{"", false},
	}
	for _, c := range cases {
		_, err := ValidateIBAN(c.in)
		if c.ok && err != nil {
			t.Errorf("expected ok for %q: %v", c.in, err)
		}
		if !c.ok && err == nil {
			t.Errorf("expected fail for %q", c.in)
		}
	}
}

func TestValidateBIC(t *testing.T) {
	cases := []struct {
		in  string
		ok  bool
	}{
		{"BNPAFRPP", true},
		{"BNPAFRPPXXX", true},
		{"", true},
		{"BAD", false},
		{"12345678", false},
	}
	for _, c := range cases {
		_, err := ValidateBIC(c.in)
		if c.ok && err != nil {
			t.Errorf("expected ok for %q: %v", c.in, err)
		}
		if !c.ok && err == nil {
			t.Errorf("expected fail for %q", c.in)
		}
	}
}
