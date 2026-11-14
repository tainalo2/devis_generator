ALTER TABLE documents
    ADD COLUMN quote_version INT NOT NULL DEFAULT 1 CHECK (quote_version > 0),
    ADD COLUMN replaces_document_id UUID REFERENCES documents(id) ON DELETE RESTRICT;

CREATE UNIQUE INDEX documents_replaces_document_id_uidx
    ON documents (replaces_document_id)
    WHERE replaces_document_id IS NOT NULL;

CREATE INDEX documents_quote_version_idx
    ON documents (id, quote_version)
    WHERE kind = 'quote';
