CREATE TABLE relance_config (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    quote_delay_days INT NOT NULL DEFAULT 7 CHECK (quote_delay_days >= 1),
    invoice_delay_days INT NOT NULL DEFAULT 14 CHECK (invoice_delay_days >= 1),
    max_relances INT NOT NULL DEFAULT 1 CHECK (max_relances BETWEEN 0 AND 3),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE relance_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    channel TEXT NOT NULL DEFAULT 'email'
);
CREATE INDEX relance_log_document_idx ON relance_log(document_id);
CREATE INDEX relance_log_user_idx ON relance_log(user_id);
EOF