CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Comptes optionnels : une ligne users = compte email/mot de passe.
-- Mode invité = documents sans user_id (pas de ligne users obligatoire).
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT,
    display_name TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    email TEXT,
    address_line1 TEXT,
    address_line2 TEXT,
    postal_code TEXT,
    city TEXT,
    country TEXT NOT NULL DEFAULT 'FR',
    siren TEXT,
    siret TEXT,
    vat_number TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX clients_user_id_idx ON clients(user_id);
CREATE INDEX clients_name_idx ON clients(name);

-- V1 : devis, facture, facture d'acompte, avoir
CREATE TYPE document_kind AS ENUM (
    'quote',
    'invoice',
    'deposit_invoice',
    'credit_note'
);

CREATE TYPE document_status AS ENUM (
    'draft',
    'sent',
    'accepted',
    'rejected',
    'paid',
    'cancelled'
);

-- franchise_293b = franchise en base ; standard = TVA collectée (taux par ligne)
CREATE TYPE vat_regime AS ENUM ('franchise_293b', 'standard');

CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    client_id UUID REFERENCES clients(id) ON DELETE SET NULL,
    kind document_kind NOT NULL,
    status document_status NOT NULL DEFAULT 'draft',
    number TEXT,
    issue_date DATE,
    due_date DATE,
    currency TEXT NOT NULL DEFAULT 'EUR',
    vat_regime vat_regime NOT NULL DEFAULT 'franchise_293b',
    notes TEXT,
    legal_mentions TEXT,
    -- Lien optionnel (acompte / avoir rattachés à une facture ou un devis)
    related_document_id UUID REFERENCES documents(id) ON DELETE SET NULL,
    subtotal_cents BIGINT NOT NULL DEFAULT 0,
    tax_cents BIGINT NOT NULL DEFAULT 0,
    total_cents BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX documents_user_id_idx ON documents(user_id);
CREATE INDEX documents_client_id_idx ON documents(client_id);
CREATE INDEX documents_kind_status_idx ON documents(kind, status);
CREATE INDEX documents_related_document_id_idx ON documents(related_document_id);
CREATE UNIQUE INDEX documents_user_number_uidx
    ON documents (user_id, number)
    WHERE number IS NOT NULL AND user_id IS NOT NULL;

CREATE TABLE document_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    position INT NOT NULL DEFAULT 0,
    description TEXT NOT NULL DEFAULT '',
    quantity NUMERIC(12, 3) NOT NULL DEFAULT 1,
    unit_price_cents BIGINT NOT NULL DEFAULT 0,
    -- Taux de TVA en points de base (ex. 2000 = 20,00 %) ; 0 en franchise
    tax_rate_bps INT NOT NULL DEFAULT 0,
    tax_cents BIGINT NOT NULL DEFAULT 0,
    total_cents BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX document_lines_document_id_idx ON document_lines(document_id);

-- Compteurs de numérotation continue par émetteur, type et année (ex. F-2026-0001).
-- issuer_key = id utilisateur (UUID) ou identifiant d'émetteur invité.
CREATE TABLE document_number_sequences (
    issuer_key TEXT NOT NULL,
    kind document_kind NOT NULL,
    year INT NOT NULL CHECK (year >= 2000 AND year <= 9999),
    prefix TEXT NOT NULL,
    last_value INT NOT NULL DEFAULT 0 CHECK (last_value >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (issuer_key, kind, year)
);

-- Cache des variables légales (TVA, seuils, pénalités, etc.).
-- Les sources officielles seront branchées plus tard via le package legal.
CREATE TABLE legal_variables (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key TEXT NOT NULL UNIQUE,
    value JSONB NOT NULL,
    source TEXT NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX legal_variables_fetched_at_idx ON legal_variables(fetched_at DESC);
