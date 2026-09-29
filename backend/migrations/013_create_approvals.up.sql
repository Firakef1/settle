-- Finance decisions on payout requests.
-- request_id is not a foreign key: the requests table belongs to the requests domain.
CREATE TABLE IF NOT EXISTS approvals (
    id VARCHAR(36) PRIMARY KEY,
    request_id VARCHAR(36) NOT NULL,
    org_id VARCHAR(36) NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    approver_id VARCHAR(36) NOT NULL REFERENCES users(id),
    decision VARCHAR(20) NOT NULL CHECK (decision IN ('approved', 'rejected')),
    decision_note TEXT,
    payment_status VARCHAR(30),
    payment_method VARCHAR(30),
    payment_at TIMESTAMP WITH TIME ZONE,
    failure_reason TEXT,
    failure_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT approvals_request_id_unique UNIQUE (request_id),
    CONSTRAINT approvals_payment_status_chk CHECK (
        payment_status IS NULL OR payment_status IN ('pending_payment', 'paid', 'failed')
    ),
    CONSTRAINT approvals_payment_method_chk CHECK (
        payment_method IS NULL OR payment_method IN ('bank_transfer', 'check', 'cash', 'other')
    ),
    CONSTRAINT approvals_decision_payment_chk CHECK (
        (decision = 'rejected' AND payment_status IS NULL)
        OR (decision = 'approved' AND payment_status IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_approvals_org_id ON approvals(org_id);
CREATE INDEX IF NOT EXISTS idx_approvals_approver_id ON approvals(approver_id);
