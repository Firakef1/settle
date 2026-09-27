package model

import "time"

// User represents the user domain model.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// OrgMembership represents a user's membership in an organization.
type OrgMembership struct {
	ID         string `json:"id"`
	OrgID      string `json:"org_id"`
	OrgName    string `json:"org_name"`
	OrgSlug    string `json:"org_slug"`
	UserID     string `json:"user_id"`
	Role       string `json:"role"`
	Department string `json:"department,omitempty"`
	Status     string `json:"status"`
}
