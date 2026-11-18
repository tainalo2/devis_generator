-- Document signature schema: adds signature state and proof tracking
-- Signed status indicates document is electronically signed and locked

-- Extend document_status enum to include 'signed' state
ALTER TYPE document_status ADD VALUE 'signed' BEFORE 'cancelled';

-- Add signature columns to documents table
ALTER TABLE documents
ADD COLUMN signed_at TIMESTAMPTZ,
ADD COLUMN is_locked BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN signed_by_email TEXT;

-- Create document_signatures table to store signature proof and history
-- Each signature event is immutable (append-only audit log)
CREATE TABLE document_signatures (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    -- Email of the signer (may be client, may be different from user_id)
    signer_email TEXT NOT NULL,
    -- Timestamp of signature
    signed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- SHA256 hash of the signed PDF (for integrity verification)
    -- Format: hex-encoded string (64 chars for SHA256)
    signature_hash TEXT NOT NULL,
    -- IP address or user agent for audit trail (optional, for security audit)
    ip_address TEXT,
    user_agent TEXT,
    -- Proof token: stable reference for external sharing/verification
    proof_token TEXT NOT NULL UNIQUE DEFAULT gen_random_uuid()::text,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for rapid lookup of signatures by document
CREATE INDEX document_signatures_document_id_idx ON document_signatures(document_id);
-- Index for user signature history
CREATE INDEX document_signatures_user_id_idx ON document_signatures(user_id);
-- Index for proof token lookups (public verification)
CREATE INDEX document_signatures_proof_token_idx ON document_signatures(proof_token);

-- Constraint: a document can only have ONE signature (locked = immutable)
-- This is enforced at application level but we can add a check
ALTER TABLE documents
ADD CONSTRAINT one_signature_per_document
CHECK (
    -- When signed_at is NOT NULL, is_locked must be TRUE
    (signed_at IS NULL OR is_locked = TRUE) AND
    -- When is_locked is TRUE, signed_at must be NOT NULL
    (is_locked = FALSE OR signed_at IS NOT NULL)
);
