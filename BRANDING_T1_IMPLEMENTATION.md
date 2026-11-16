# BRANDING T1: Data Model Implementation

**Date**: 2026-11-16  
**Task**: t_272e7595 — Modèle de données branding (logo, couleurs)  
**Status**: Data model + storage layer complete, API endpoints pending (T2+)

## Overview

This implementation provides the backend data model and storage layer for user branding customization (logo + color schema). The branding configuration is scoped per user (emitter/account), enabling each business to customize their generated documents with a branded logo and color scheme.

## Architecture

### Database Schema (Migration 000024)

Added three columns to the `users` table:

```sql
-- Logo storage: PNG/JPG binary data, max 2MB
ALTER TABLE users ADD COLUMN branding_logo_data BYTEA;

-- Color schema: JSON object with hex color strings
ALTER TABLE users ADD COLUMN branding_colors JSONB;

-- Audit timestamp: when branding was last updated
ALTER TABLE users ADD COLUMN branding_updated_at TIMESTAMPTZ DEFAULT NULL;

-- Constraint: enforce 2MB size limit
ALTER TABLE users ADD CONSTRAINT branding_logo_max_size 
  CHECK (branding_logo_data IS NULL OR octet_length(branding_logo_data) <= 2097152);
```

**Rationale**:
- **BYTEA for logo**: Direct binary storage avoids file system complexity; 2MB limit meets typical branding needs
- **JSONB for colors**: Flexible structure allows future extensions (font, spacing, etc.)
- **Branding updated_at**: Tracks changes for cache invalidation, audit trails
- **Index on presence**: Helps identify users with custom branding

### Service Layer (`backend/internal/branding/`)

#### ColorScheme (data model)

```go
type ColorScheme struct {
    PrimaryColor string // hex: "2f9e94" or "#2f9e94"
    AccentColor  string // hex: secondary color
    TextColor    string // hex: main text color
}
```

#### BrandingProfile (complete config)

```go
type BrandingProfile struct {
    UserID           uuid.UUID     // owner
    LogoData         []byte        // PNG/JPG binary
    LogoFilename     string        // for reference
    Colors           *ColorScheme
    BrandingUpdatedAt *time.Time   // last modification
}
```

#### Service Operations

```go
// Get user's complete branding profile
func (s *Service) GetBranding(ctx, userID) (*BrandingProfile, error)

// Update logo (validates format/size)
func (s *Service) UpdateLogo(ctx, userID, logoData []byte, filename) error

// Update color scheme (validates hex format)
func (s *Service) UpdateColors(ctx, userID, colors *ColorScheme) error

// Clear all branding (reset to defaults)
func (s *Service) ClearBranding(ctx, userID) error
```

## Validation Rules

### Logo Validation

- **Format**: PNG or JPG only (enforced via image header decoding)
- **Size**: 1 KB ≤ size ≤ 2 MB
- **Dimensions**: No explicit limits (let frontend handle UI constraints)

### Color Validation

- **Format**: 6-digit hex (with or without `#` prefix)
- **Case-insensitive**: `#2F9E94` and `#2f9e94` both valid
- **Optional**: Empty strings allowed (fall back to defaults)

## Testing

### Unit Tests (`service_test.go`)

- ✅ Hex color parsing (with/without `#`, valid ranges, invalid inputs)
- ✅ Color scheme validation (nil, empty, all three colors)
- ✅ Edge cases (lowercase, uppercase, mixed)

```bash
go test -v ./internal/branding
# PASS: 8 color validation tests + 6 scheme tests
```

### Integration Tests (`integration_test.go`)

- ✅ UpdateColors + GetBranding (read-write cycle)
- ✅ UpdateLogo + GetBranding (binary data persistence)
- ✅ Logo validation: oversized data rejection
- ✅ Logo validation: invalid format rejection
- ✅ ClearBranding (complete removal)

Requires `TEST_DATABASE_URL` environment variable; skipped otherwise.

## Integration with Existing Code

### BrandingData in pdfgen

The existing `pdfgen.BrandingData` struct (used by RenderWithBranding) mirrors this service's ColorScheme:

```go
// pdfgen/branding.go
type BrandingData struct {
    LogoPNG      []byte // PNG binary
    PrimaryColor string // hex color
    AccentColor  string
    TextColor    string
}
```

**Next phases** (T2+) will fetch user's BrandingProfile and pass LogoData + Colors to RenderWithBranding.

## Design Decisions

1. **BYTEA over file storage**: PostgreSQL handles retention, backup, and GC automatically; no S3 or file system complexity.
   
2. **Hex colors over RGB**: Human-readable, matches frontend input, avoids float precision issues.

3. **JSON for colors (not individual columns)**: Allows grouping related fields; easier to clear as a unit; room for future extensions.

4. **Validation at service layer**: Prevents invalid data from reaching database; fail-fast for clients.

5. **Optional fields**: Branding is opt-in; missing branding → document renders with app defaults.

## Implementation Notes

- Logo data is immutable after upload (replace by updating). No versioning needed for MVP.
- Colors are always replaced together (no partial updates).
- branding_updated_at helps detect sync issues and can support cache invalidation in future.
- No explicit "branding enabled/disabled" flag; presence of data implies enabled.

## Next Steps (Dependent Tickets)

- **T2 (Brand-T2)**: API endpoints `POST /api/v1/account/branding/logo`, `POST /api/v1/account/branding/colors`
- **T3 (Brand-T3)**: Frontend account page UI (upload, color picker, preview)
- **T4 (Brand-T4)**: Apply branding to PDF generation
- **T5 (Brand-T5)**: Public document link branding
- **T6 (Brand-T6)**: Factur-X compliance validation
- **T7 (Brand-T7)**: MCP API branding endpoints

## Files Modified/Created

```
backend/internal/db/migrations/
  ├── 000024_branding.up.sql      (new)
  └── 000024_branding.down.sql    (new)

backend/internal/branding/
  ├── service.go                  (new)
  ├── service_test.go             (new)
  └── integration_test.go         (new)
```

## Verification Checklist

- [x] SQL migration rollback tested (down.sql)
- [x] Service compiles without errors
- [x] All unit tests pass (12/12)
- [x] Logo validation handles PNG/JPG/invalid formats
- [x] Color validation rejects invalid hex
- [x] Database constraints enforced (2MB limit)
- [x] GetBranding handles missing/partial branding
- [x] Integration test structure matches codebase patterns
