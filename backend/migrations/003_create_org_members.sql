CREATE TABLE IF NOT EXISTS org_members (
    org_id    UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id   UUID NOT NULL,
    role      VARCHAR(50) NOT NULL CHECK (role IN ('org_admin', 'staff', 'finance')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (org_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_org_members_user_id ON org_members(user_id);
