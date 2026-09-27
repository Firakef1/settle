package dto

import "time"

type MemberResponse struct {
	UserID   string    `json:"user_id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required"`
}
