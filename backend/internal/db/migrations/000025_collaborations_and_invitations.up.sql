-- Collaborations and Invitations System
-- Supports account owner sharing access with other users via invitation tokens.
-- Each collaborator has a role (owner, admin, editor, viewer).

-- Table: account_collaborators
-- Links users to accounts with their role.
-- An account is owned by a user; additional collaborators can have read/write/admin roles.
CREATE TABLE account_collaborators (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- The account owner (the primary user of the account)
    account_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- The collaborator being granted access
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- Role: owner (account owner), admin (full access except account deletion), editor (can modify documents/clients), viewer (read-only)
    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'editor', 'viewer')),
    -- When the collaborator was added
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- One collaborator relationship per account/user pair
    UNIQUE(account_id, user_id)
);

CREATE INDEX account_collaborators_account_id_idx ON account_collaborators(account_id);
CREATE INDEX account_collaborators_user_id_idx ON account_collaborators(user_id);
CREATE INDEX account_collaborators_role_idx ON account_collaborators(role);

-- Table: invitations
-- Pending invitations to join an account.
-- Each invitation is single-use and expires after a configurable period (default: 7 days).
CREATE TABLE invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- The account being shared
    account_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- Email address of the invited person (may not yet be a user)
    email TEXT NOT NULL,
    -- Role to be granted if invitation is accepted
    role TEXT NOT NULL CHECK (role IN ('admin', 'editor', 'viewer')),
    -- Single-use token for accepting the invitation
    -- Token format: random 32-byte hex string, stored unhashed (invitations are short-lived and email-verified)
    token TEXT NOT NULL UNIQUE,
    -- Invitation expiration time
    expires_at TIMESTAMPTZ NOT NULL,
    -- When the invitation was created
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- The user who created the invitation (for audit)
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    -- One active invitation per (account, email) pair at a time
    -- Multiple invitations to the same email can exist if previous ones have expired
    UNIQUE(account_id, email) WHERE expires_at > NOW()
);

CREATE INDEX invitations_account_id_idx ON invitations(account_id);
CREATE INDEX invitations_email_idx ON invitations(email);
CREATE INDEX invitations_token_idx ON invitations(token);
CREATE INDEX invitations_expires_at_idx ON invitations(expires_at);
CREATE INDEX invitations_created_by_idx ON invitations(created_by);
