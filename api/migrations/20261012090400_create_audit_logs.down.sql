BEGIN;

DROP TABLE audit_logs;
DROP FUNCTION audit_logs_prevent_change();

COMMIT;
