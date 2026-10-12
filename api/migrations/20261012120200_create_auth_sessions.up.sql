-- Sign-in sessions (1.01, docs/AUTHENTICATION.md). One row per sign-in on a device; the refresh
-- tokens of a session rotate in auth_refresh_tokens. Revoking the session ends every token.
BEGIN;

CREATE TABLE auth_sessions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    remember     BOOLEAN NOT NULL DEFAULT false,
    platform     TEXT NOT NULL DEFAULT 'unknown',
    user_agent   TEXT,
    ip_address   INET,
    expires_at   TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT auth_sessions_platform_check   CHECK (platform IN ('web', 'android', 'ios', 'unknown')),
    CONSTRAINT auth_sessions_user_agent_check CHECK (user_agent IS NULL OR length(user_agent) <= 512),
    CONSTRAINT auth_sessions_expiry_check     CHECK (expires_at > created_at)
);

-- Active sessions of a user ("sign out everywhere", the future "Active sessions" list).
CREATE INDEX auth_sessions_user_id_idx ON auth_sessions (user_id) WHERE revoked_at IS NULL;

CREATE TRIGGER auth_sessions_set_updated_at BEFORE UPDATE ON auth_sessions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Refresh tokens: SHA-256 of the opaque token. A used token is kept so its reuse (theft) can be
-- detected and the whole session revoked.
CREATE TABLE auth_refresh_tokens (
    token_hash TEXT PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES auth_sessions (id) ON DELETE CASCADE,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT auth_refresh_tokens_hash_check CHECK (length(token_hash) = 64)
);

CREATE INDEX auth_refresh_tokens_session_id_idx ON auth_refresh_tokens (session_id);

COMMIT;
