-- Payment secrets at rest and immutable access audit.
CREATE TABLE IF NOT EXISTS payment_secrets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('iban', 'byo_provider_api_key', 'stripe_connect_credential')),
    reference TEXT NOT NULL CHECK (char_length(reference) BETWEEN 1 AND 128),
    key_id SMALLINT NOT NULL CHECK (key_id BETWEEN 1 AND 255),
    ciphertext BYTEA NOT NULL,
    masked_value TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (owner_id, kind, reference)
);

CREATE INDEX IF NOT EXISTS payment_secrets_owner_idx ON payment_secrets(owner_id, kind);

CREATE TABLE IF NOT EXISTS payment_secret_audit (
    id BIGSERIAL PRIMARY KEY,
    secret_id UUID REFERENCES payment_secrets(id) ON DELETE SET NULL,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    reference TEXT NOT NULL,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_request_id TEXT,
    action TEXT NOT NULL CHECK (action IN ('create', 'read', 'update', 'delete')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS payment_secret_audit_owner_idx ON payment_secret_audit(owner_id, created_at DESC);
CREATE INDEX IF NOT EXISTS payment_secret_audit_secret_idx ON payment_secret_audit(secret_id, created_at DESC);
