package branding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidLogoFormat   = errors.New("logo must be PNG or JPG")
	ErrLogoTooLarge        = errors.New("logo exceeds 2MB limit")
	ErrInvalidHexColor     = errors.New("invalid hex color format")
	ErrBrandingNotFound    = errors.New("branding configuration not found")
	ErrNoLogoProvided      = errors.New("no logo provided for update")
	ErrNoBrandingProvided  = errors.New("no branding data provided")
)

const (
	MaxLogoSize = 2 * 1024 * 1024 // 2MB
	MinLogoSize = 1024            // 1KB minimum (avoid empty uploads)
)

// ColorScheme holds the three user-selected colors for branding.
// All fields are optional — if nil, default styling is used.
type ColorScheme struct {
	PrimaryColor string `json:"primary_color,omitempty"`  // hex: "2f9e94" or "#2f9e94"
	AccentColor  string `json:"accent_color,omitempty"`   // hex: secondary color
	TextColor    string `json:"text_color,omitempty"`     // hex: main text color
}

// BrandingProfile is the complete branding configuration for a user.
type BrandingProfile struct {
	UserID           uuid.UUID     `json:"user_id"`
	LogoData         []byte        `json:"logo_data,omitempty"` // PNG/JPG binary
	LogoFilename     string        `json:"logo_filename,omitempty"`
	Colors           *ColorScheme  `json:"colors"`
	BrandingUpdatedAt *time.Time   `json:"branding_updated_at,omitempty"`
}

// Service manages user branding data.
type Service struct {
	pool *pgxpool.Pool
}

// New creates a new branding service.
func New(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// ValidateLogoData checks logo format and size constraints.
func ValidateLogoData(data []byte, filename string) error {
	if len(data) == 0 {
		return ErrNoLogoProvided
	}
	if len(data) < MinLogoSize || len(data) > MaxLogoSize {
		return ErrLogoTooLarge
	}

	// Check if it's valid PNG or JPG by decoding image header
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidLogoFormat, err)
	}
	if format != "png" && format != "jpeg" {
		return fmt.Errorf("%w: got %s", ErrInvalidLogoFormat, format)
	}

	return nil
}

// ValidateColor checks hex color format.
func ValidateColor(hex string) error {
	if hex == "" {
		return nil // empty is allowed (optional)
	}
	// Allow with or without # prefix
	if hex[0] == '#' {
		hex = hex[1:]
	}
	if !regexp.MustCompile(`^[0-9a-fA-F]{6}$`).MatchString(hex) {
		return ErrInvalidHexColor
	}
	return nil
}

// ValidateColorScheme checks all colors in the scheme.
func ValidateColorScheme(cs *ColorScheme) error {
	if cs == nil {
		return nil
	}
	if err := ValidateColor(cs.PrimaryColor); err != nil {
		return fmt.Errorf("primary color: %w", err)
	}
	if err := ValidateColor(cs.AccentColor); err != nil {
		return fmt.Errorf("accent color: %w", err)
	}
	if err := ValidateColor(cs.TextColor); err != nil {
		return fmt.Errorf("text color: %w", err)
	}
	return nil
}

// GetBranding retrieves a user's branding profile.
func (s *Service) GetBranding(ctx context.Context, userID uuid.UUID) (*BrandingProfile, error) {
	var profile BrandingProfile
	var colorsJSON []byte

	err := s.pool.QueryRow(ctx, `
		SELECT user_id, branding_logo_data, branding_colors, branding_updated_at
		FROM users WHERE id = $1
	`, userID).Scan(&profile.UserID, &profile.LogoData, &colorsJSON, &profile.BrandingUpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch branding: %w", err)
	}

	if colorsJSON != nil {
		if err := json.Unmarshal(colorsJSON, &profile.Colors); err != nil {
			return nil, fmt.Errorf("failed to parse colors: %w", err)
		}
	}

	return &profile, nil
}

// UpdateLogo updates a user's logo, validating format and size.
func (s *Service) UpdateLogo(ctx context.Context, userID uuid.UUID, logoData []byte, filename string) error {
	if err := ValidateLogoData(logoData, filename); err != nil {
		return err
	}

	_, err := s.pool.Exec(ctx, `
		UPDATE users 
		SET branding_logo_data = $2, branding_updated_at = NOW()
		WHERE id = $1
	`, userID, logoData)

	return err
}

// UpdateColors updates a user's color scheme, validating hex values.
func (s *Service) UpdateColors(ctx context.Context, userID uuid.UUID, colors *ColorScheme) error {
	if colors == nil {
		return ErrNoBrandingProvided
	}
	if err := ValidateColorScheme(colors); err != nil {
		return err
	}

	colorsJSON, err := json.Marshal(colors)
	if err != nil {
		return fmt.Errorf("failed to marshal colors: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE users 
		SET branding_colors = $2, branding_updated_at = NOW()
		WHERE id = $1
	`, userID, colorsJSON)

	return err
}

// ClearBranding removes all branding for a user.
func (s *Service) ClearBranding(ctx context.Context, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE users 
		SET branding_logo_data = NULL, branding_colors = NULL, branding_updated_at = NULL
		WHERE id = $1
	`, userID)

	return err
}
