DROP TABLE IF EXISTS reminder_delivery_attempts;
DROP INDEX IF EXISTS reminders_rate_limit_idx;

DELETE FROM reminders r
USING reminders duplicate
WHERE r.user_id = duplicate.user_id
  AND r.idempotency_key = duplicate.idempotency_key
  AND (r.created_at, r.id) < (duplicate.created_at, duplicate.id);
ALTER TABLE reminders DROP CONSTRAINT reminders_delivery_idempotency_key;
ALTER TABLE reminders ADD CONSTRAINT reminders_user_id_idempotency_key_key
    UNIQUE (user_id, idempotency_key);

ALTER TABLE reminders DROP CONSTRAINT reminders_document_tenant_fk;
ALTER TABLE reminders DROP CONSTRAINT reminders_template_tenant_fk;
ALTER TABLE reminders
    ADD CONSTRAINT reminders_invoice_id_fkey
        FOREIGN KEY (invoice_id) REFERENCES documents(id) ON DELETE CASCADE,
    ADD CONSTRAINT reminders_template_id_fkey
        FOREIGN KEY (template_id) REFERENCES reminder_templates(id) ON DELETE RESTRICT;

UPDATE reminders SET status='failed', lease_until=NULL
WHERE status IN ('sending','skipped');
ALTER TABLE reminders DROP CONSTRAINT reminders_status_check;
ALTER TABLE reminders ADD CONSTRAINT reminders_status_check
    CHECK (status IN ('queued','sent','failed','bounced'));
ALTER TABLE reminders DROP COLUMN attempt_count, DROP COLUMN lease_until;

ALTER TABLE documents DROP CONSTRAINT documents_user_id_id_key;
ALTER TABLE reminder_templates DROP CONSTRAINT reminder_templates_user_id_id_key;
DROP TABLE IF EXISTS email_delivery_settings;
