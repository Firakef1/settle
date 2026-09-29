package dto

import "time"

// ApprovalResponse is the finance decision returned to the client.
type ApprovalResponse struct {
	ID            string     `json:"id"`
	RequestID     string     `json:"request_id"`
	OrgID         string     `json:"org_id"`
	ApproverID    string     `json:"approver_id"`
	Decision      string     `json:"decision"`
	DecisionNote  string     `json:"decision_note,omitempty"`
	PaymentStatus string     `json:"payment_status,omitempty"`
	PaymentMethod string     `json:"payment_method,omitempty"`
	PaymentAt     *time.Time `json:"payment_at,omitempty"`
	FailureReason string     `json:"failure_reason,omitempty"`
	FailureAt     *time.Time `json:"failure_at,omitempty"`
	RequestStatus string     `json:"request_status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
