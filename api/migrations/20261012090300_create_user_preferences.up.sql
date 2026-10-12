-- Appearance settings that follow the account across devices (PRD §4.4, §4.6, §21).
-- Values match the web theme attributes from 0.02 (docs/ARCHITECTURE.md §4.1). A missing row
-- means "all defaults". Wallpaper (2.11) and notification settings (2.11/3.14) are added by
-- their sessions as new columns.
BEGIN;

CREATE TABLE user_preferences (
    user_id           UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    theme             TEXT NOT NULL DEFAULT 'light',
    accent            TEXT NOT NULL DEFAULT 'neutral',
    density           TEXT NOT NULL DEFAULT 'comfortable',
    motion            TEXT NOT NULL DEFAULT 'system',
    sidebar_collapsed BOOLEAN NOT NULL DEFAULT false,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT user_preferences_theme_check   CHECK (theme IN ('light', 'dark', 'system')),
    CONSTRAINT user_preferences_accent_check
        CHECK (accent IN ('neutral', 'blue', 'indigo', 'violet', 'teal', 'rose')),
    CONSTRAINT user_preferences_density_check CHECK (density IN ('comfortable', 'compact')),
    CONSTRAINT user_preferences_motion_check  CHECK (motion IN ('system', 'reduced', 'full'))
);

CREATE TRIGGER user_preferences_set_updated_at BEFORE UPDATE ON user_preferences
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
