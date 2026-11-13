CREATE TABLE email_delivery_settings (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    mode TEXT NOT NULL CHECK (mode IN ('gmail','microsoft','smtp')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE reminder_templates
    ADD CONSTRAINT reminder_templates_user_id_id_key UNIQUE (user_id, id);
ALTER TABLE documents
    ADD CONSTRAINT documents_user_id_id_key UNIQUE (user_id, id);

ALTER TABLE reminders
    ADD COLUMN lease_until TIMESTAMPTZ,
    ADD COLUMN attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0);

ALTER TABLE reminders DROP CONSTRAINT reminders_status_check;
ALTER TABLE reminders ADD CONSTRAINT reminders_status_check
    CHECK (status IN ('queued','sending','sent','failed','bounced','skipped'));

ALTER TABLE reminders DROP CONSTRAINT reminders_invoice_id_fkey;
ALTER TABLE reminders DROP CONSTRAINT reminders_template_id_fkey;
ALTER TABLE reminders
    ADD CONSTRAINT reminders_document_tenant_fk
        FOREIGN KEY (user_id, invoice_id) REFERENCES documents(user_id, id) ON DELETE CASCADE,
    ADD CONSTRAINT reminders_template_tenant_fk
        FOREIGN KEY (user_id, template_id) REFERENCES reminder_templates(user_id, id) ON DELETE RESTRICT;

ALTER TABLE reminders DROP CONSTRAINT reminders_user_id_idempotency_key_key;
ALTER TABLE reminders ADD CONSTRAINT reminders_delivery_idempotency_key
    UNIQUE (user_id, invoice_id, template_id, reminder_type, idempotency_key);

CREATE INDEX reminders_rate_limit_idx ON reminders(user_id, created_at DESC)
    WHERE status IN ('queued','sending','sent');

CREATE TABLE reminder_delivery_attempts (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    reminder_id UUID NOT NULL REFERENCES reminders(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    attempted_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX reminder_delivery_attempts_rate_idx
    ON reminder_delivery_attempts(user_id, attempted_at DESC);
