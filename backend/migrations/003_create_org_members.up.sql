-- +up
CREATE TABLE IF NOT EXISTS org_members (
    id VARCHAR(36) PRIMARY KEY,
    org_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'staff',
    department VARCHAR(100),
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(org_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_org_members_user_id ON org_members(user_id);
CREATE INDEX IF NOT EXISTS idx_org_members_org_id ON org_members(org_id);
