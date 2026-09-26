package dto

// LogoutRequest represents the request payload for logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
