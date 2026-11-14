ALTER TABLE payment_secrets DROP CONSTRAINT IF EXISTS payment_secrets_kind_check;
ALTER TABLE payment_secrets ADD CONSTRAINT payment_secrets_kind_check CHECK (kind IN ('iban','bic','byo_provider_api_key','stripe_connect_credential','oauth_gmail_token','oauth_microsoft_token','smtp_password','stripe_api_key','stripe_webhook_secret','mollie_api_key','paypal_client_id','paypal_client_secret','paypal_webhook_id'));

CREATE TABLE payment_provider_configs (
 owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 provider TEXT NOT NULL CHECK(provider IN ('stripe','mollie','paypal','stripe_connect')),
 environment TEXT NOT NULL CHECK(environment IN ('sandbox','live')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 PRIMARY KEY(owner_id,provider)
);
CREATE INDEX payment_provider_configs_owner_idx ON payment_provider_configs(owner_id);

CREATE TABLE stripe_connect_accounts (
 owner_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 account_id TEXT NOT NULL UNIQUE, country CHAR(2) NOT NULL, default_currency CHAR(3) NOT NULL DEFAULT 'EUR',
	capabilities JSONB NOT NULL DEFAULT '{}'::jsonb, compliance JSONB NOT NULL DEFAULT '{}'::jsonb,
 charges_enabled BOOLEAN NOT NULL DEFAULT FALSE, payouts_enabled BOOLEAN NOT NULL DEFAULT FALSE,
 card_payments_active BOOLEAN NOT NULL DEFAULT FALSE, transfers_active BOOLEAN NOT NULL DEFAULT FALSE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX stripe_connect_accounts_account_idx ON stripe_connect_accounts(account_id);

CREATE UNIQUE INDEX IF NOT EXISTS documents_user_id_id_idx ON documents(user_id,id);

CREATE TABLE payment_attempts (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 document_id UUID NOT NULL, provider TEXT NOT NULL, external_id TEXT, checkout_url TEXT, idempotency_key TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'creating' CHECK(status IN ('creating','pending','paid','failed')), last_error TEXT,
	reserved_at TIMESTAMPTZ, lease_expires_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE(provider,idempotency_key), UNIQUE(owner_id,id),
 FOREIGN KEY(owner_id,document_id) REFERENCES documents(user_id,id) ON DELETE CASCADE
);
CREATE INDEX payment_attempts_owner_document_idx ON payment_attempts(owner_id,document_id);
CREATE UNIQUE INDEX payment_attempts_provider_external_idx ON payment_attempts(provider,external_id) WHERE external_id IS NOT NULL;

CREATE TABLE payment_webhook_events (
 id BIGSERIAL PRIMARY KEY, provider TEXT NOT NULL, event_id TEXT NOT NULL,
 owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, payment_attempt_id UUID NOT NULL,
 received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE(provider,event_id),
 FOREIGN KEY(owner_id,payment_attempt_id) REFERENCES payment_attempts(owner_id,id) ON DELETE CASCADE
);
CREATE INDEX payment_webhook_events_owner_idx ON payment_webhook_events(owner_id,received_at DESC);
