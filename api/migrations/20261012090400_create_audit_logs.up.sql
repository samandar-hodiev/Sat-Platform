-- Append-only record of important Owner (and security) actions (PRD §21.1, §28).
-- Written through the audit helper from 1.02; read by the Audit Logs page (4.07).
BEGIN;

CREATE TABLE audit_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- NULL for system actions, and after the actor's account is deleted.
    actor_user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    -- Role at the time of the action, so the entry stays readable after the account is gone.
    actor_role    TEXT,
    -- "<entity>.<verb>", e.g. question.publish, learner.suspend, owner.sign_in.
    action        TEXT NOT NULL,
    entity_type   TEXT,
    -- Text, not UUID: some entities use natural keys (plan codes, setting names).
    entity_id     TEXT,
    -- Small structured context (changed fields, old/new status). Never secrets or tokens.
    metadata      JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip_address    INET,
    user_agent    TEXT,
    request_id    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT audit_logs_actor_role_check
        CHECK (actor_role IS NULL OR actor_role ~ '^[a-z][a-z0-9_]{0,31}$'),
    CONSTRAINT audit_logs_action_check
        CHECK (action ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$' AND length(action) <= 100),
    CONSTRAINT audit_logs_entity_type_check
        CHECK (entity_type IS NULL OR entity_type ~ '^[a-z][a-z0-9_]{0,63}$'),
    CONSTRAINT audit_logs_entity_id_check
        CHECK (entity_id IS NULL OR length(entity_id) BETWEEN 1 AND 128),
    CONSTRAINT audit_logs_metadata_check
        CHECK (jsonb_typeof(metadata) = 'object'),
    CONSTRAINT audit_logs_user_agent_check
        CHECK (user_agent IS NULL OR length(user_agent) <= 512),
    CONSTRAINT audit_logs_request_id_check
        CHECK (request_id IS NULL OR length(request_id) <= 128)
);

-- Filters on the Audit Logs page (4.07) and "recent admin actions" (4.02).
CREATE INDEX audit_logs_created_at_idx ON audit_logs (created_at DESC);
CREATE INDEX audit_logs_actor_user_id_created_at_idx ON audit_logs (actor_user_id, created_at DESC);
CREATE INDEX audit_logs_entity_type_entity_id_created_at_idx
    ON audit_logs (entity_type, entity_id, created_at DESC);
CREATE INDEX audit_logs_action_created_at_idx ON audit_logs (action, created_at DESC);

-- Entries are never changed or removed. The single allowed UPDATE is the foreign key's
-- ON DELETE SET NULL clearing actor_user_id when a user is deleted; anything else fails.
CREATE FUNCTION audit_logs_prevent_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    -- Nested IFs: OLD/NEW do not exist in the TRUNCATE (statement-level) trigger.
    IF TG_OP = 'UPDATE' THEN
        IF OLD.actor_user_id IS NOT NULL
           AND NEW.actor_user_id IS NULL
           AND (to_jsonb(NEW) - 'actor_user_id') = (to_jsonb(OLD) - 'actor_user_id') THEN
            RETURN NEW;
        END IF;
    END IF;
    RAISE EXCEPTION 'audit_logs is append-only (% is not allowed)', TG_OP
        USING ERRCODE = 'insufficient_privilege';
END;
$$;

CREATE TRIGGER audit_logs_prevent_update_delete BEFORE UPDATE OR DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION audit_logs_prevent_change();
CREATE TRIGGER audit_logs_prevent_truncate BEFORE TRUNCATE ON audit_logs
    FOR EACH STATEMENT EXECUTE FUNCTION audit_logs_prevent_change();

COMMIT;
