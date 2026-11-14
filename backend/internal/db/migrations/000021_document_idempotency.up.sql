ALTER TABLE documents
    ADD COLUMN idempotency_key TEXT;

CREATE UNIQUE INDEX documents_user_id_idempotency_key_uidx
    ON documents (user_id, idempotency_key)
    WHERE user_id IS NOT NULL AND idempotency_key IS NOT NULL;
