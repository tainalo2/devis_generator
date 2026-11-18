-- Rollback document signature schema
DROP INDEX IF EXISTS document_signatures_proof_token_idx;
DROP INDEX IF EXISTS document_signatures_user_id_idx;
DROP INDEX IF EXISTS document_signatures_document_id_idx;
DROP TABLE IF EXISTS document_signatures;

-- Remove signature columns from documents
ALTER TABLE documents
DROP CONSTRAINT IF EXISTS one_signature_per_document,
DROP COLUMN IF EXISTS signed_by_email,
DROP COLUMN IF EXISTS is_locked,
DROP COLUMN IF EXISTS signed_at;

-- Remove 'signed' status from document_status enum
-- Note: PostgreSQL does not support removing values from an ENUM,
-- so we create a new enum, migrate data, and replace it
ALTER TYPE document_status RENAME TO document_status_old;

CREATE TYPE document_status AS ENUM (
    'draft',
    'sent',
    'accepted',
    'rejected',
    'paid',
    'cancelled'
);

ALTER TABLE documents
ALTER COLUMN status TYPE document_status
USING CASE status::text
    WHEN 'draft' THEN 'draft'::document_status
    WHEN 'sent' THEN 'sent'::document_status
    WHEN 'accepted' THEN 'accepted'::document_status
    WHEN 'rejected' THEN 'rejected'::document_status
    WHEN 'paid' THEN 'paid'::document_status
    WHEN 'signed' THEN 'sent'::document_status  -- Map signed back to sent
    WHEN 'cancelled' THEN 'cancelled'::document_status
    ELSE 'draft'::document_status
END;

DROP TYPE document_status_old;
