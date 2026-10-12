-- Shared trigger function that keeps updated_at current on every mutable table.
-- Usage in later migrations:
--   CREATE TRIGGER <table>_set_updated_at BEFORE UPDATE ON <table>
--       FOR EACH ROW EXECUTE FUNCTION set_updated_at();
BEGIN;

CREATE FUNCTION set_updated_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$;

COMMIT;
