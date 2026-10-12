-- External sign-in identities (1.01). Google accounts are matched by the stable subject ("sub"),
-- never by email, so a learner keeps their account if their Google email changes.
BEGIN;

CREATE TABLE user_identities (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider   TEXT NOT NULL,
    subject    TEXT NOT NULL,
    -- Email reported by the provider when the identity was linked (informational).
    email      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT user_identities_provider_check CHECK (provider IN ('google')),
    CONSTRAINT user_identities_subject_check  CHECK (length(subject) BETWEEN 1 AND 255)
);

CREATE UNIQUE INDEX user_identities_provider_subject_key ON user_identities (provider, subject);
CREATE INDEX user_identities_user_id_idx ON user_identities (user_id);

CREATE TRIGGER user_identities_set_updated_at BEFORE UPDATE ON user_identities
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
