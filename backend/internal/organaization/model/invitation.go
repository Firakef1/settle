package model

import "time"

type Invitation struct {
	ID        string     `json:"id" db:"id"`
	OrgID     string     `json:"org_id" db:"org_id"`
	Email     string     `json:"email" db:"email"`
	Role      string     `json:"role" db:"role"` // staff | finance
	Token     string     `json:"token" db:"token"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}
