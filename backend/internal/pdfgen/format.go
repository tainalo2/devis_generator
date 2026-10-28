package pdfgen

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tainalo2/devis_generator/backend/internal/biz"
)

// nbsp sépare les milliers et précède le symbole €.
// U+00A0 est dans la police (le U+202F étroit de fr-FR moderne n’y est pas) :
// le montant ne se coupe pas, et le glyphe s’affiche.
const nbsp = "\u00a0"

func formatEUR(cents int64) string {
	neg := cents < 0
	if neg {
		cents = -cents
	}
	s := groupInt(cents/100) + "," + twoDigits(cents%100) + nbsp + "€"
	if neg {
		return "\u2212" + s
	}
	return s
}

func twoDigits(n int64) string {
	if n < 10 {
		return "0" + strconv.FormatInt(n, 10)
	}
	return strconv.FormatInt(n, 10)
}

func groupInt(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	lead := len(s) % 3
	if lead == 0 {
		lead = 3
	}
	var b strings.Builder
	b.WriteString(s[:lead])
	for i := lead; i < len(s); i += 3 {
		b.WriteString(nbsp)
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// formatQty affiche des milli-unités comme le ferait fr-FR (1,5 et non 1,500).
func formatQty(milli int64) string {
	neg := milli < 0
	if neg {
		milli = -milli
	}
	s := groupInt(milli / 1000)
	if frac := milli % 1000; frac != 0 {
		fs := strings.TrimRight(strconv.FormatInt(frac+1000, 10)[1:], "0")
		s += "," + fs
	}
	if neg {
		return "\u2212" + s
	}
	return s
}

func formatRate(bps int) string {
	neg := bps < 0
	if neg {
		bps = -bps
	}
	whole := bps / 100
	frac := bps % 100
	s := strconv.Itoa(whole)
	if frac != 0 {
		fs := strconv.Itoa(frac + 100)[1:]
		s += "," + strings.TrimRight(fs, "0")
	}
	s += " %"
	if neg {
		return "\u2212" + s
	}
	return s
}

func formatDate(iso string) string {
	iso = strings.TrimSpace(iso)
	if iso == "" {
		return "\u2014"
	}
	if t, err := time.Parse("2006-01-02", iso); err == nil {
		return t.Format("02/01/2006")
	}
	if len(iso) >= 10 {
		if t, err := time.Parse("2006-01-02", iso[:10]); err == nil {
			return t.Format("02/01/2006")
		}
	}
	return iso
}

// clean garde un texte UTF-8 affichable : pas de contrôles, pas de caractères
// hors du plan multilingue de base (la carte de largeurs de la police s’arrête à U+FFFF).
func clean(s string) string {
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "")
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			if r == '\t' {
				return ' '
			}
			return r
		}
		if r < 32 {
			return -1
		}
		if r > 0xFFFF {
			return '?'
		}
		return r
	}, s)
}

func addressLines(a biz.Address) []string {
	var out []string
	if s := strings.TrimSpace(a.Line1); s != "" {
		out = append(out, s)
	}
	if s := strings.TrimSpace(a.Line2); s != "" {
		out = append(out, s)
	}
	city := strings.TrimSpace(strings.TrimSpace(a.PostalCode) + " " + strings.TrimSpace(a.City))
	if city != "" {
		out = append(out, city)
	}
	if c := strings.TrimSpace(a.Country); c != "" && !strings.EqualFold(c, "France") && !strings.EqualFold(c, "FR") {
		out = append(out, c)
	}
	return out
}
