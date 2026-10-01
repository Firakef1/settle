package model

import "time"

type Request struct {
	ID            string     `json:"id" db:"id"`
	OrgID         string     `json:"org_id" db:"org_id"`
	RequesterID   string     `json:"requester_id" db:"requester_id"`
	Type          string     `json:"type" db:"type"`
	Amount        float64    `json:"amount" db:"amount"`
	Purpose       string     `json:"purpose" db:"purpose"`
	Urgency       string     `json:"urgency" db:"urgency"`
	Status        string     `json:"status" db:"status"`
	ResubmittedAs *string    `json:"resubmitted_as,omitempty" db:"resubmitted_as"`
	SubmittedAt   *time.Time `json:"submitted_at,omitempty" db:"submitted_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}
