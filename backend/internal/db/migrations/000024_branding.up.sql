-- Add branding columns to users table
-- Logo storage: PNG/JPG image data (max 2MB per constraint)
-- Colors storage: JSON object with primary_color, accent_color, text_color (hex strings)

ALTER TABLE users ADD COLUMN branding_logo_data BYTEA;
ALTER TABLE users ADD COLUMN branding_colors JSONB;

-- Add constraint to prevent oversized logos (2MB = 2097152 bytes)
ALTER TABLE users ADD CONSTRAINT branding_logo_max_size 
  CHECK (branding_logo_data IS NULL OR octet_length(branding_logo_data) <= 2097152);

-- Index for potential future queries filtering by branding presence
CREATE INDEX users_has_branding_idx ON users 
  WHERE branding_logo_data IS NOT NULL OR branding_colors IS NOT NULL;

-- Create a simple audit: when was branding last updated per user
-- This helps track changes and detect sync issues
ALTER TABLE users ADD COLUMN branding_updated_at TIMESTAMPTZ DEFAULT NULL;
