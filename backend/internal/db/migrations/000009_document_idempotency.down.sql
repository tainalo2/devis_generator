DROP INDEX IF EXISTS documents_user_id_idempotency_key_uidx;

ALTER TABLE documents
    DROP COLUMN IF EXISTS idempotency_key;
