-- Profil émetteur (identité légale pour PDF) + champs documents émis.

CREATE TABLE IF NOT EXISTS issuer_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    legal_name TEXT NOT NULL DEFAULT '',
    trade_name TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    siret TEXT NOT NULL DEFAULT '',
    siren TEXT NOT NULL DEFAULT '',
    vat_number TEXT NOT NULL DEFAULT '',
    ape_code TEXT NOT NULL DEFAULT '',
    rcs TEXT NOT NULL DEFAULT '',
    address_enc BYTEA,
    iban_enc BYTEA,
    default_vat_regime vat_regime NOT NULL DEFAULT 'franchise_293b',
    payment_terms TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE documents ADD COLUMN IF NOT EXISTS issued_at TIMESTAMPTZ;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS issuer_snapshot JSONB;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS legal_snapshot JSONB;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS payment_terms TEXT NOT NULL DEFAULT '';
ALTER TABLE documents ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT '';

-- Statut « issued » = document numéroté immuable (si absent).
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_enum e
        JOIN pg_type t ON t.oid = e.enumtypid
        WHERE t.typname = 'document_status' AND e.enumlabel = 'issued'
    ) THEN
        ALTER TYPE document_status ADD VALUE 'issued';
    END IF;
END$$;

ALTER TABLE document_lines ADD COLUMN IF NOT EXISTS discount_bps INT NOT NULL DEFAULT 0;
ALTER TABLE document_lines ADD COLUMN IF NOT EXISTS quantity_milli BIGINT;
UPDATE document_lines SET quantity_milli = ROUND(quantity * 1000)::BIGINT WHERE quantity_milli IS NULL;
ALTER TABLE document_lines ALTER COLUMN quantity_milli SET DEFAULT 1000;
ALTER TABLE document_lines ALTER COLUMN quantity_milli SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS documents_user_kind_number_uidx
    ON documents (user_id, kind, number)
    WHERE number IS NOT NULL AND user_id IS NOT NULL;
