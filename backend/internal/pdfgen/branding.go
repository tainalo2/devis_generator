package pdfgen

import "fmt"

// BrandingData holds user branding configuration (logo, colors).
// All fields are optional — if not provided, default styling is used.
type BrandingData struct {
	// LogoPNG is the PNG-encoded logo image, or nil if not provided.
	// Max 2MB, PNG/JPG format.
	LogoPNG []byte

	// PrimaryColor is the hex color string (e.g., "#2f9e94" or "2f9e94").
	// Used for headings, table headers, accents.
	PrimaryColor string

	// AccentColor is a secondary accent color (hex string).
	// Used for highlights, borders.
	AccentColor string

	// TextColor is the main text color (hex string).
	// Used for body text, secondary text.
	TextColor string
}

// IsEmpty returns true if no branding data is configured.
func (b BrandingData) IsEmpty() bool {
	return len(b.LogoPNG) == 0 && b.PrimaryColor == "" && b.AccentColor == "" && b.TextColor == ""
}

// parseHexColor converts a hex color string (e.g., "2f9e94" or "#2f9e94") to RGB components.
// Returns (R, G, B, error). Values are in [0, 255].
func parseHexColor(hex string) (int, int, int, error) {
	// Remove leading '#' if present
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}

	// Must be exactly 6 hex digits
	if len(hex) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid color format: expected 6 hex digits, got %d", len(hex))
	}

	var r, g, b int
	_, err := scanHex(hex[0:2], &r)
	if err != nil {
		return 0, 0, 0, err
	}
	_, err = scanHex(hex[2:4], &g)
	if err != nil {
		return 0, 0, 0, err
	}
	_, err = scanHex(hex[4:6], &b)
	if err != nil {
		return 0, 0, 0, err
	}

	return r, g, b, nil
}

// scanHex scans two hex digits and stores the value in *v.
func scanHex(s string, v *int) (string, error) {
	if len(s) < 2 {
		return "", fmt.Errorf("invalid color format: too short")
	}
	for _, c := range s[:2] {
		var d int
		switch {
		case '0' <= c && c <= '9':
			d = int(c - '0')
		case 'a' <= c && c <= 'f':
			d = int(c - 'a' + 10)
		case 'A' <= c && c <= 'F':
			d = int(c - 'A' + 10)
		default:
			return "", fmt.Errorf("invalid color format: non-hex character")
		}
		*v = *v*16 + d
	}
	return s[2:], nil
}
