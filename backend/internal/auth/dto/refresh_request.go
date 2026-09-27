package dto

// RefreshRequest represents the request payload for token refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
