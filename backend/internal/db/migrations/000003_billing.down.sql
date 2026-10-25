DROP INDEX IF EXISTS documents_user_kind_number_uidx;

ALTER TABLE document_lines DROP COLUMN IF EXISTS quantity_milli;
ALTER TABLE document_lines DROP COLUMN IF EXISTS discount_bps;

ALTER TABLE documents DROP COLUMN IF EXISTS title;
ALTER TABLE documents DROP COLUMN IF EXISTS payment_terms;
ALTER TABLE documents DROP COLUMN IF EXISTS legal_snapshot;
ALTER TABLE documents DROP COLUMN IF EXISTS issuer_snapshot;
ALTER TABLE documents DROP COLUMN IF EXISTS issued_at;

DROP TABLE IF EXISTS issuer_profiles;
-- Note: PostgreSQL ne permet pas de retirer facilement une valeur d'enum ('issued').
