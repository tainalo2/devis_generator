// Package bankcoord manages owner bank coordinates (IBAN/BIC) stored
// encrypted at rest through the paymentsecrets vault.
package bankcoord

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidIBAN = errors.New("invalid IBAN")
	ErrInvalidBIC  = errors.New("invalid BIC/SWIFT")
	ErrMissingName = errors.New("account holder name required")
)

var bicRe = regexp.MustCompile(`^[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?$`)

// NormalizeIBAN strips spaces and upper-cases.
func NormalizeIBAN(s string) string {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	return strings.ToUpper(strings.TrimSpace(s))
}

// ValidateIBAN checks format and mod-97 checksum (ISO 13616).
func ValidateIBAN(raw string) (string, error) {
	iban := NormalizeIBAN(raw)
	if len(iban) < 15 || len(iban) > 34 {
		return "", ErrInvalidIBAN
	}
	// first two letters country, two digits check
	for i := 0; i < 2; i++ {
		if iban[i] < 'A' || iban[i] > 'Z' {
			return "", ErrInvalidIBAN
		}
	}
	for i := 2; i < 4; i++ {
		if iban[i] < '0' || iban[i] > '9' {
			return "", ErrInvalidIBAN
		}
	}
	for i := 4; i < len(iban); i++ {
		c := iban[i]
		if !(c >= '0' && c <= '9') && !(c >= 'A' && c <= 'Z') {
			return "", ErrInvalidIBAN
		}
	}
	// mod 97
	rearr := iban[4:] + iban[:4]
	num := ""
	for _, c := range rearr {
		if c >= 'A' && c <= 'Z' {
			num += itoa(int(c-'A') + 10)
		} else {
			num += string(c)
		}
	}
	if mod97(num) != 1 {
		return "", ErrInvalidIBAN
	}
	return iban, nil
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	// two digits
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

func mod97(num string) int {
	rem := 0
	for _, c := range num {
		rem = (rem*10 + int(c-'0')) % 97
	}
	return rem
}

// ValidateBIC validates optional BIC/SWIFT.
func ValidateBIC(raw string) (string, error) {
	bic := strings.ToUpper(strings.TrimSpace(raw))
	if bic == "" {
		return "", nil
	}
	if !bicRe.MatchString(bic) {
		return "", ErrInvalidBIC
	}
	return bic, nil
}
