-- Bank coordinates (IBAN/BIC) per issuer, encrypted via payment_secrets vault.
-- This table stores metadata only; the actual IBAN/BIC are in payment_secrets.
CREATE TABLE IF NOT EXISTS bank_coordinates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_holder_name TEXT NOT NULL CHECK (char_length(account_holder_name) BETWEEN 1 AND 255),
    -- Reference to the payment_secrets vault entry (iban kind)
    secret_id UUID NOT NULL REFERENCES payment_secrets(id) ON DELETE CASCADE,
    bic_secret_id UUID REFERENCES payment_secrets(id) ON DELETE SET NULL,
    bank_name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id)
);

CREATE INDEX IF NOT EXISTS bank_coordinates_user_idx ON bank_coordinates(user_id);
CREATE INDEX IF NOT EXISTS bank_coordinates_secret_idx ON bank_coordinates(secret_id);
