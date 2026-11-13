ALTER TABLE documents
    ADD COLUMN reminder_count INTEGER NOT NULL DEFAULT 0 CHECK (reminder_count >= 0),
    ADD COLUMN last_reminder_at TIMESTAMPTZ;

CREATE TABLE reminder_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    subject TEXT NOT NULL,
    html TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX reminder_templates_user_idx ON reminder_templates(user_id);

CREATE TABLE reminders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    template_id UUID NOT NULL REFERENCES reminder_templates(id) ON DELETE RESTRICT,
    reminder_type TEXT NOT NULL CHECK (reminder_type IN ('manual','scheduled','job')),
    recipient TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued','sent','failed','bounced')),
    idempotency_key TEXT NOT NULL CHECK (char_length(idempotency_key) BETWEEN 1 AND 200),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMPTZ,
    UNIQUE (user_id, idempotency_key)
);
CREATE INDEX reminders_history_idx ON reminders(user_id, invoice_id, created_at DESC, id DESC);
