package facturx

import (
	"fmt"
	"math/big"
	"strings"
	"time"
	"unicode"
)

func formatCents(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

// formatQty écrit des milli-unités (1000 = 1) sans zéros inutiles.
func formatQty(milli int64) string {
	sign := ""
	if milli < 0 {
		sign = "-"
		milli = -milli
	}
	whole := milli / 1000
	frac := milli % 1000
	if frac == 0 {
		return fmt.Sprintf("%s%d", sign, whole)
	}
	fs := strings.TrimRight(fmt.Sprintf("%03d", frac), "0")
	return fmt.Sprintf("%s%d.%s", sign, whole, fs)
}

func formatPercent(bps int) string {
	sign := ""
	if bps < 0 {
		sign = "-"
		bps = -bps
	}
	return fmt.Sprintf("%s%d.%02d", sign, bps/100, bps%100)
}

// formatUnitEuros écrit un prix unitaire (centimes / quantité en milli-unités).
func formatUnitEuros(cents, qtyMilli int64) string {
	if qtyMilli == 0 {
		return formatCents(cents)
	}
	r := new(big.Rat).SetFrac(big.NewInt(cents*10), big.NewInt(qtyMilli))
	return formatRat(r)
}

func formatRat(r *big.Rat) string {
	s := r.FloatString(6)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	if i := strings.IndexByte(s, '.'); i >= 0 {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	if !strings.Contains(s, ".") {
		s += ".00"
	} else if len(s)-strings.IndexByte(s, '.')-1 == 1 {
		s += "0"
	}
	if neg && s != "0.00" {
		return "-" + s
	}
	return s
}

func parseDate(iso string) (time.Time, error) {
	iso = strings.TrimSpace(iso)
	if len(iso) >= 10 {
		if t, err := time.Parse("2006-01-02", iso[:10]); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("date d'émission invalide")
}

func digits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// sirenOf retourne le SIREN (9 chiffres) depuis le SIREN ou, à défaut, le SIRET.
func sirenOf(siren, siret string) string {
	s := digits(siren)
	if len(s) >= 9 {
		return s[:9]
	}
	t := digits(siret)
	if len(t) >= 9 {
		return t[:9]
	}
	return ""
}

func siretOf(siret string) string {
	t := digits(siret)
	if len(t) >= 14 {
		return t[:14]
	}
	return ""
}

func compactIBAN(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func countryCode(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, "’", "'")
	if s == "" {
		return "FR", nil
	}
	if code, ok := countries[s]; ok {
		return code, nil
	}
	if len(s) == 2 && isAlpha(s) {
		return strings.ToUpper(s), nil
	}
	return "", fmt.Errorf("pays inconnu : %s", strings.TrimSpace(raw))
}

func isAlpha(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return s != ""
}

var countries = map[string]string{
	"fr": "FR", "fra": "FR", "france": "FR",
	"be": "BE", "bel": "BE", "belgique": "BE", "belgium": "BE",
	"de": "DE", "deu": "DE", "allemagne": "DE", "germany": "DE",
	"es": "ES", "esp": "ES", "espagne": "ES", "spain": "ES",
	"it": "IT", "ita": "IT", "italie": "IT", "italy": "IT",
	"lu": "LU", "lux": "LU", "luxembourg": "LU",
	"ch": "CH", "che": "CH", "suisse": "CH", "switzerland": "CH",
	"gb": "GB", "gbr": "GB", "uk": "GB", "royaume-uni": "GB", "royaume uni": "GB",
	"nl": "NL", "nld": "NL", "pays-bas": "NL", "pays bas": "NL", "netherlands": "NL",
}
