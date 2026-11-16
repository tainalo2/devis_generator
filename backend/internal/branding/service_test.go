package branding

import (
	"testing"
)

func TestValidateColor(t *testing.T) {
	tests := []struct {
		name    string
		hex     string
		wantErr bool
	}{
		{"valid hex without #", "2f9e94", false},
		{"valid hex with #", "#2f9e94", false},
		{"empty string allowed", "", false},
		{"invalid short", "2f9e9", true},
		{"invalid long", "2f9e94ff", true},
		{"invalid chars", "2f9e9g", true},
		{"lowercase valid", "#abc123", false},
		{"uppercase valid", "#ABC123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateColor(tt.hex)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateColor(%q) error = %v, wantErr %v", tt.hex, err, tt.wantErr)
			}
		})
	}
}

func TestValidateColorScheme(t *testing.T) {
	tests := []struct {
		name    string
		scheme  *ColorScheme
		wantErr bool
	}{
		{"nil scheme", nil, false},
		{"empty scheme", &ColorScheme{}, false},
		{"valid scheme", &ColorScheme{PrimaryColor: "2f9e94", AccentColor: "#FF0000", TextColor: "000000"}, false},
		{"invalid primary", &ColorScheme{PrimaryColor: "invalid"}, true},
		{"invalid accent", &ColorScheme{AccentColor: "12345"}, true},
		{"invalid text", &ColorScheme{TextColor: "gggggg"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateColorScheme(tt.scheme)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateColorScheme(%+v) error = %v, wantErr %v", tt.scheme, err, tt.wantErr)
			}
		})
	}
}

// Logo validation requires actual PNG/JPG data, tested in integration_test.go
