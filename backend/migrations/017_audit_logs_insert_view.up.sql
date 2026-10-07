-- Requests insert into audit_logs (id, org_id, actor_id, action, target_type, target_id, created_at).
-- The real table is audit_log, which has no target_type column. This view accepts that
-- insert and stores target_type inside metadata so GET /organizations/:id/audit-log
-- returns the row.

CREATE OR REPLACE VIEW audit_logs AS
SELECT
    id,
    org_id,
    actor_id,
    action,
    metadata->>'target_type' AS target_type,
    target_id,
    created_at
FROM audit_log;

CREATE OR REPLACE FUNCTION audit_logs_insert()
RETURNS trigger
LANGUAGE plpgsql
SET search_path = public
AS $$
BEGIN
    INSERT INTO audit_log (id, org_id, actor_id, action, target_id, metadata, created_at)
    VALUES (
        NEW.id,
        NEW.org_id,
        NEW.actor_id,
        NEW.action,
        NEW.target_id,
        jsonb_build_object('target_type', NEW.target_type),
        COALESCE(NEW.created_at, CURRENT_TIMESTAMP)
    );
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS audit_logs_instead_of_insert ON audit_logs;

CREATE TRIGGER audit_logs_instead_of_insert
    INSTEAD OF INSERT ON audit_logs
    FOR EACH ROW
    EXECUTE FUNCTION audit_logs_insert();
