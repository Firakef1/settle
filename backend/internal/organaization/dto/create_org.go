package dto

import "time"

type CreateOrgRequest struct {
	Name     string `json:"name" binding:"required"`
	Currency string `json:"currency" binding:"required"`
}

type OrgResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Currency  string    `json:"currency"`
	Plan      string    `json:"plan"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
