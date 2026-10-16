-- One-time codes for signed-in account changes (2.10, PRD §20 Account): confirming the learner
-- before a sensitive change ("reauth", sent to the current email) and proving a new address
-- ("email_change", sent to the new email). Kept apart from email_verification_codes, which are
-- keyed by email for sign-up, so a code of one flow can never be spent in another. Hashes are
-- HMAC-SHA256 bound to purpose and email; the plain code exists only in the email.
BEGIN;

CREATE TABLE account_action_codes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    purpose    TEXT NOT NULL,
    -- Where the code was sent: the current email (reauth) or the requested new one (email_change).
    email      TEXT NOT NULL,
    code_hash  TEXT NOT NULL,
    attempts   INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT account_action_codes_purpose_check  CHECK (purpose IN ('reauth', 'email_change')),
    CONSTRAINT account_action_codes_email_check    CHECK (email = lower(btrim(email)) AND position('@' IN email) > 1),
    CONSTRAINT account_action_codes_hash_check     CHECK (length(code_hash) = 64),
    CONSTRAINT account_action_codes_attempts_check CHECK (attempts >= 0),
    CONSTRAINT account_action_codes_expiry_check   CHECK (expires_at > created_at)
);

-- Newest code of a user and purpose (verify), and codes sent in the last 24 h (cooldown, cap).
CREATE INDEX account_action_codes_user_purpose_created_at_idx
    ON account_action_codes (user_id, purpose, created_at DESC);

COMMIT;
