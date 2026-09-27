package model

import "time"

type OrgMember struct {
	OrgID    string    `json:"org_id" db:"org_id"`
	UserID   string    `json:"user_id" db:"user_id"`
	Role     string    `json:"role" db:"role"` // org_admin | staff | finance
	JoinedAt time.Time `json:"joined_at" db:"joined_at"`
}
