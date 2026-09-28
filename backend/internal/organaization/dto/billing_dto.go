package dto

import "time"

// Plan represents a subscription tier offered by the platform.
type Plan struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Price        float64  `json:"price"`
	RequestLimit int      `json:"request_limit"` // 0 represents unlimited
	UserLimit    int      `json:"user_limit"`    // 0 represents unlimited
	Features     []string `json:"features,omitempty"`
}

// UpdatePlanRequest is the request payload for updating an organization's subscription plan.
type UpdatePlanRequest struct {
	Plan string `json:"plan" binding:"required"`
}

// UpdatePlanResponse is the response returned after successfully updating a plan.
type UpdatePlanResponse struct {
	OrgID     string    `json:"org_id"`
	Plan      string    `json:"plan"`
	UpdatedAt time.Time `json:"updated_at"`
}
