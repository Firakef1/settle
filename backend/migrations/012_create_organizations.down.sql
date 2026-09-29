DROP TABLE IF EXISTS invitations;
DROP TABLE IF EXISTS audit_log;
ALTER TABLE org_members DROP COLUMN IF EXISTS joined_at;
DROP TABLE IF EXISTS organizations;
