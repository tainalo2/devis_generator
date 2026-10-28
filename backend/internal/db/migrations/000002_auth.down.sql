ALTER TABLE clients DROP COLUMN IF EXISTS address_enc;
ALTER TABLE clients DROP COLUMN IF EXISTS iban_enc;

ALTER TABLE documents DROP CONSTRAINT IF EXISTS documents_user_id_fkey;
ALTER TABLE documents
    ADD CONSTRAINT documents_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE clients DROP CONSTRAINT IF EXISTS clients_user_id_fkey;
ALTER TABLE clients
    ADD CONSTRAINT clients_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS sessions;
