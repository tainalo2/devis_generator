-- Signature électronique simple horodatée (eIDAS art. 3(10)), voir docs/DECISIONS.md.
-- Une seule signature par document (UNIQUE) ; la ligne est en ajout seul.
CREATE TABLE document_signatures (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id         UUID NOT NULL UNIQUE REFERENCES documents(id) ON DELETE CASCADE,
    signed_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    document_hash       TEXT NOT NULL CHECK (document_hash ~ '^[0-9a-f]{64}$'),
    signer_name         TEXT NOT NULL CHECK (char_length(btrim(signer_name)) BETWEEN 2 AND 200),
    signer_ip           TEXT NOT NULL DEFAULT '',
    signer_user_agent   TEXT NOT NULL DEFAULT ''
);

CREATE FUNCTION document_signatures_no_update() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'document_signatures is append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER document_signatures_no_update
    BEFORE UPDATE ON document_signatures
    FOR EACH ROW EXECUTE FUNCTION document_signatures_no_update();
