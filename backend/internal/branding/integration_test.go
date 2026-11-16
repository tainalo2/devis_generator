package branding

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"testing"
	"time"

	"bytes"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tainalo2/devis_generator/backend/internal/db"
)

func setupBrandingTestDB(t *testing.T) (*pgxpool.Pool, context.Context, uuid.UUID) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	if err := db.RunMigrations(dsn); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	userID := uuid.New()
	email := fmt.Sprintf("branding-%s@example.test", userID)
	if _, err = pool.Exec(ctx, `INSERT INTO users(id,email,display_name) VALUES($1,$2,'branding test')`, userID, email); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID) })
	return pool, ctx, userID
}

func TestIntegrationBrandingOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, ctx, userID := setupBrandingTestDB(t)
	service := New(pool)

	t.Run("UpdateColors", func(t *testing.T) {
		colors := &ColorScheme{
			PrimaryColor: "2f9e94",
			AccentColor:  "#ff0000",
			TextColor:    "000000",
		}

		err := service.UpdateColors(ctx, userID, colors)
		if err != nil {
			t.Fatalf("UpdateColors failed: %v", err)
		}

		profile, err := service.GetBranding(ctx, userID)
		if err != nil {
			t.Fatalf("GetBranding failed: %v", err)
		}

		if profile.Colors.PrimaryColor != colors.PrimaryColor {
			t.Errorf("primary color mismatch: got %q, want %q", profile.Colors.PrimaryColor, colors.PrimaryColor)
		}
	})

	t.Run("UpdateLogo", func(t *testing.T) {
		// Create a minimal valid PNG
		logoData := createTestPNG(100, 100)

		err := service.UpdateLogo(ctx, userID, logoData, "test.png")
		if err != nil {
			t.Fatalf("UpdateLogo failed: %v", err)
		}

		profile, err := service.GetBranding(ctx, userID)
		if err != nil {
			t.Fatalf("GetBranding failed: %v", err)
		}

		if len(profile.LogoData) != len(logoData) {
			t.Errorf("logo size mismatch: got %d, want %d", len(profile.LogoData), len(logoData))
		}
	})

	t.Run("LogoValidation_TooLarge", func(t *testing.T) {
		oversizedData := make([]byte, MaxLogoSize+1)
		err := service.UpdateLogo(ctx, userID, oversizedData, "toolarge.png")
		if err == nil {
			t.Error("expected error for oversized logo, got nil")
		}
	})

	t.Run("LogoValidation_InvalidFormat", func(t *testing.T) {
		invalidData := []byte("not an image")
		err := service.UpdateLogo(ctx, userID, invalidData, "invalid.txt")
		if err == nil {
			t.Error("expected error for invalid format, got nil")
		}
	})

	t.Run("ClearBranding", func(t *testing.T) {
		// First set some branding
		colors := &ColorScheme{PrimaryColor: "aabbcc"}
		err := service.UpdateColors(ctx, userID, colors)
		if err != nil {
			t.Fatalf("UpdateColors failed: %v", err)
		}

		// Clear it
		err = service.ClearBranding(ctx, userID)
		if err != nil {
			t.Fatalf("ClearBranding failed: %v", err)
		}

		// Verify it's gone
		profile, err := service.GetBranding(ctx, userID)
		if err != nil {
			t.Fatalf("GetBranding failed: %v", err)
		}

		if profile.LogoData != nil || profile.Colors != nil {
			t.Errorf("branding not cleared: logo=%v, colors=%v", profile.LogoData, profile.Colors)
		}
	})
}

// createTestPNG creates a minimal valid PNG image of given dimensions.
func createTestPNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	buf := &bytes.Buffer{}
	if err := png.Encode(buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
