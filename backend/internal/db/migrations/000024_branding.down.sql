-- Rollback branding columns from users table

ALTER TABLE users DROP CONSTRAINT IF EXISTS branding_logo_max_size;
DROP INDEX IF EXISTS users_has_branding_idx;
ALTER TABLE users DROP COLUMN IF EXISTS branding_updated_at;
ALTER TABLE users DROP COLUMN IF EXISTS branding_colors;
ALTER TABLE users DROP COLUMN IF EXISTS branding_logo_data;
