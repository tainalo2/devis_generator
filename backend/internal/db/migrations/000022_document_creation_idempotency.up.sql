-- Idempotence durable des créations de documents (en-tête Idempotency-Key).
-- Une ligne par (compte, opération, clé). La clé est insérée AVANT la création
-- du document, dans la même transaction : la contrainte PRIMARY KEY sérialise
-- les requêtes concurrentes portant la même clé. request_hash est le SHA-256
-- de la requête normalisée : même clé + hash différent => conflit (409).
-- Les lignes expirées (expires_at, 24 h après création) sont purgées.
CREATE TABLE document_idempotency_keys (
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operation       TEXT        NOT NULL CHECK (char_length(operation) BETWEEN 1 AND 64),
    idempotency_key TEXT        NOT NULL CHECK (char_length(idempotency_key) BETWEEN 1 AND 255),
    request_hash    TEXT        NOT NULL CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    document_id     UUID        REFERENCES documents(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, operation, idempotency_key)
);

CREATE INDEX document_idempotency_keys_expires_at_idx
    ON document_idempotency_keys (expires_at);
