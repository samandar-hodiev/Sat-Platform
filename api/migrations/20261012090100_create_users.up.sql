-- Accounts for learners and the Owner. Sign-in data only: names and contact details live in
-- profiles, appearance in user_preferences. Auth tables (sessions, codes, identities) are
-- added by 1.01 and reference users(id).
BEGIN;

CREATE TABLE users (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Normalised by the API (trim + lowercase); the constraint guards against a missed path.
    email             TEXT NOT NULL,
    -- Argon2id PHC string. NULL for accounts that only sign in with Google.
    password_hash     TEXT,
    email_verified_at TIMESTAMPTZ,
    role              TEXT NOT NULL DEFAULT 'learner',
    status            TEXT NOT NULL DEFAULT 'active',
    last_login_at     TIMESTAMPTZ,
    -- Set when the account is deleted (2.10); the row is kept for history until purged.
    deleted_at        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT users_email_normalized_check
        CHECK (email = lower(btrim(email)) AND length(email) BETWEEN 3 AND 254 AND position('@' IN email) > 1),
    CONSTRAINT users_password_hash_check
        CHECK (password_hash IS NULL OR length(password_hash) BETWEEN 1 AND 512),
    CONSTRAINT users_role_check   CHECK (role IN ('learner', 'owner')),
    CONSTRAINT users_status_check CHECK (status IN ('active', 'suspended'))
);

-- One live account per email; a deleted account frees its email.
CREATE UNIQUE INDEX users_email_key ON users (email) WHERE deleted_at IS NULL;
-- Owner lists and counts learners by registration date (4.02, 4.03).
CREATE INDEX users_role_created_at_idx ON users (role, created_at DESC) WHERE deleted_at IS NULL;

CREATE TRIGGER users_set_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
