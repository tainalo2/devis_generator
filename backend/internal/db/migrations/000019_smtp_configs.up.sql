CREATE TABLE smtp_configs (
    owner_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    host TEXT NOT NULL CHECK (char_length(host) BETWEEN 1 AND 253),
    port INTEGER NOT NULL CHECK (port BETWEEN 1 AND 65535),
    username TEXT NOT NULL CHECK (char_length(username) BETWEEN 1 AND 254),
    from_address TEXT NOT NULL CHECK (char_length(from_address) BETWEEN 3 AND 254),
    tls_mode TEXT NOT NULL CHECK (tls_mode IN ('starttls', 'tls')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
