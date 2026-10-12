-- Email verification codes for sign-up / sign-in by code (1.01, PRD §6.1). Codes are stored as
-- HMAC-SHA256 hashes bound to the email; the plain code exists only in the email. Rows are keyed
-- by email, not user: the account does not exist until the code is verified.
BEGIN;

CREATE TABLE email_verification_codes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email      TEXT NOT NULL,
    code_hash  TEXT NOT NULL,
    attempts   INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT email_verification_codes_email_check    CHECK (email = lower(btrim(email))),
    CONSTRAINT email_verification_codes_hash_check     CHECK (length(code_hash) = 64),
    CONSTRAINT email_verification_codes_attempts_check CHECK (attempts >= 0),
    CONSTRAINT email_verification_codes_expiry_check   CHECK (expires_at > created_at)
);

-- Newest code per email (verify) and codes sent in the last 24 h (resend cooldown, daily cap).
CREATE INDEX email_verification_codes_email_created_at_idx
    ON email_verification_codes (email, created_at DESC);

COMMIT;
