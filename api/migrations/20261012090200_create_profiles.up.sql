-- Personal details of a user (PRD §7). One row per user, created with an upsert by the
-- session that first has data for it (Google sign-up in 1.01 or Profile Setup in 2.01).
-- The email is not repeated here: users.email is the only copy.
BEGIN;

CREATE TABLE profiles (
    user_id              UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    first_name           TEXT,
    last_name            TEXT,
    -- E.164, e.g. +998901234567. Optional.
    phone                TEXT,
    date_of_birth        DATE,
    -- Object storage key (avatars/<user-id>/<uuid>.webp). NULL means the default avatar.
    avatar_key           TEXT,
    -- IANA name (Asia/Tashkent). Used for "today" in study plans and reminder times.
    timezone             TEXT NOT NULL DEFAULT 'UTC',
    -- Set when Profile Setup is finished; onboarding starts after it.
    profile_completed_at TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT profiles_first_name_check
        CHECK (first_name IS NULL OR length(btrim(first_name)) BETWEEN 1 AND 100),
    CONSTRAINT profiles_last_name_check
        CHECK (last_name IS NULL OR length(btrim(last_name)) BETWEEN 1 AND 100),
    CONSTRAINT profiles_phone_check
        CHECK (phone IS NULL OR phone ~ '^\+[1-9][0-9]{6,14}$'),
    CONSTRAINT profiles_date_of_birth_check
        CHECK (date_of_birth IS NULL OR date_of_birth >= DATE '1900-01-01'),
    CONSTRAINT profiles_avatar_key_check
        CHECK (avatar_key IS NULL OR length(avatar_key) BETWEEN 1 AND 512),
    CONSTRAINT profiles_timezone_check
        CHECK (length(timezone) BETWEEN 1 AND 64)
);

CREATE TRIGGER profiles_set_updated_at BEFORE UPDATE ON profiles
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
