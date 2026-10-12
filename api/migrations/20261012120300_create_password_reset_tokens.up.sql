-- Password reset links (1.01). SHA-256 of a 256-bit token; single use, short-lived. A new request
-- supersedes the user's earlier unused tokens (marked used).
BEGIN;

CREATE TABLE password_reset_tokens (
    token_hash TEXT PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT password_reset_tokens_hash_check   CHECK (length(token_hash) = 64),
    CONSTRAINT password_reset_tokens_expiry_check CHECK (expires_at > created_at)
);

CREATE INDEX password_reset_tokens_user_id_idx ON password_reset_tokens (user_id) WHERE used_at IS NULL;

COMMIT;
