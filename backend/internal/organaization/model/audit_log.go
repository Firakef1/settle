package model

import "time"

type AuditLog struct {
	ID        string    `json:"id" db:"id"`
	OrgID     string    `json:"org_id" db:"org_id"`
	ActorID   string    `json:"actor_id" db:"actor_id"`
	Action    string    `json:"action" db:"action"` // org_created | member_removed | role_updated | invite_created | invite_accepted
	TargetID  *string   `json:"target_id,omitempty" db:"target_id"`
	Metadata  string    `json:"metadata" db:"metadata"` // JSON string
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
