DROP INDEX IF EXISTS documents_quote_version_idx;
DROP INDEX IF EXISTS documents_replaces_document_id_uidx;

ALTER TABLE documents
    DROP COLUMN IF EXISTS replaces_document_id,
    DROP COLUMN IF EXISTS quote_version;
