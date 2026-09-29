package model

import (
	"database/sql"
	"time"
)

const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
	StatusPaid     = "paid"
	StatusFailed   = "failed"

	DecisionApproved = "approved"
	DecisionRejected = "rejected"

	PaymentPending = "pending_payment"
	PaymentPaid    = "paid"
	PaymentFailed  = "failed"

	ActionApproved = "approved"
	ActionRejected = "rejected"
	ActionPaid     = "paid"
	ActionFailed   = "failed"

	RoleFinance  = "finance"
	RoleOrgAdmin = "org_admin"
)

// Approval is one finance decision for a payout request.
type Approval struct {
	ID            string
	RequestID     string
	OrgID         string
	ApproverID    string
	Decision      string
	DecisionNote  sql.NullString
	PaymentStatus sql.NullString
	PaymentMethod sql.NullString
	PaymentAt     sql.NullTime
	FailureReason sql.NullString
	FailureAt     sql.NullTime
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// AuditEntry is written to audit_log inside the approval transaction.
type AuditEntry struct {
	ID        string
	OrgID     string
	ActorID   string
	Action    string
	TargetID  string
	Metadata  string
	CreatedAt time.Time
}

// RequestSnapshot is the request state this domain is allowed to read.
type RequestSnapshot struct {
	ID     string
	OrgID  string
	Status string
}
