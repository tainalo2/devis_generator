ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS payment_feature_flags (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
    rib_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    byo_provider_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    stripe_connect_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO payment_feature_flags (singleton)
VALUES (TRUE)
ON CONFLICT (singleton) DO NOTHING;
