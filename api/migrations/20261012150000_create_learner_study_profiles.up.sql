-- Onboarding answers (session 2.02, PRD §8): the learner's goal, study schedule and background.
-- One row per learner, written step by step; each step has its own *_saved_at so onboarding
-- resumes where the learner stopped. completed_at marks the end of onboarding (then Placement,
-- PRD §9). The answers stay editable afterwards (Settings → Learning Preferences, 2.11) and are
-- the inputs of the study plan (2.05). NULL means "not answered", never a default.
BEGIN;

CREATE TABLE learner_study_profiles (
    user_id              UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,

    -- Step "goal" (§8.1). exam_date NULL with goal_saved_at set: the learner has not decided.
    target_score         SMALLINT,
    exam_date            DATE,
    reason               TEXT,
    goal_saved_at        TIMESTAMPTZ,

    -- Step "schedule" (§8.2). study_days: ISO weekdays, 1 = Monday … 7 = Sunday, sorted.
    daily_minutes        SMALLINT,
    study_days           SMALLINT[],
    schedule_saved_at    TIMESTAMPTZ,

    -- Step "background" (§8.2). Confidence 1 (not confident) … 5 (very confident).
    math_confidence      SMALLINT,
    rw_confidence        SMALLINT,
    -- An earlier official SAT result, if any. Sections are optional; given together and summing
    -- to the total. taken_on is the first day of the (approximate) month.
    prev_total_score     SMALLINT,
    prev_math_score      SMALLINT,
    prev_rw_score        SMALLINT,
    prev_taken_on        DATE,
    background_saved_at  TIMESTAMPTZ,

    completed_at         TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT learner_study_profiles_target_score_check
        CHECK (target_score IS NULL OR (target_score BETWEEN 400 AND 1600 AND target_score % 10 = 0)),
    CONSTRAINT learner_study_profiles_reason_check
        CHECK (reason IS NULL OR reason IN
            ('college_admission', 'scholarship', 'improve_score', 'practice', 'other')),
    CONSTRAINT learner_study_profiles_goal_check
        CHECK (goal_saved_at IS NULL OR target_score IS NOT NULL),
    CONSTRAINT learner_study_profiles_daily_minutes_check
        CHECK (daily_minutes IS NULL OR daily_minutes IN (15, 30, 45, 60, 90, 120)),
    CONSTRAINT learner_study_profiles_study_days_check
        CHECK (study_days IS NULL OR (cardinality(study_days) BETWEEN 1 AND 7
            AND study_days <@ ARRAY[1, 2, 3, 4, 5, 6, 7]::SMALLINT[])),
    CONSTRAINT learner_study_profiles_schedule_check
        CHECK (schedule_saved_at IS NULL OR (daily_minutes IS NOT NULL AND study_days IS NOT NULL)),
    CONSTRAINT learner_study_profiles_confidence_check
        CHECK ((math_confidence IS NULL OR math_confidence BETWEEN 1 AND 5)
            AND (rw_confidence IS NULL OR rw_confidence BETWEEN 1 AND 5)),
    CONSTRAINT learner_study_profiles_background_check
        CHECK (background_saved_at IS NULL OR (math_confidence IS NOT NULL AND rw_confidence IS NOT NULL)),
    CONSTRAINT learner_study_profiles_prev_total_check
        CHECK (prev_total_score IS NULL OR (prev_total_score BETWEEN 400 AND 1600 AND prev_total_score % 10 = 0)),
    CONSTRAINT learner_study_profiles_prev_sections_check
        CHECK ((prev_math_score IS NULL AND prev_rw_score IS NULL)
            OR (prev_total_score IS NOT NULL
                AND prev_math_score IS NOT NULL AND prev_rw_score IS NOT NULL
                AND prev_math_score BETWEEN 200 AND 800 AND prev_math_score % 10 = 0
                AND prev_rw_score BETWEEN 200 AND 800 AND prev_rw_score % 10 = 0
                AND prev_math_score + prev_rw_score = prev_total_score)),
    CONSTRAINT learner_study_profiles_prev_taken_on_check
        CHECK (prev_taken_on IS NULL OR (prev_total_score IS NOT NULL
            AND extract(day FROM prev_taken_on) = 1 AND prev_taken_on >= DATE '2016-03-01')),
    CONSTRAINT learner_study_profiles_completed_check
        CHECK (completed_at IS NULL OR (goal_saved_at IS NOT NULL
            AND schedule_saved_at IS NOT NULL AND background_saved_at IS NOT NULL))
);

CREATE TRIGGER learner_study_profiles_set_updated_at BEFORE UPDATE ON learner_study_profiles
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
