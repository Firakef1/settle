package dto

// DeleteAccountRequest is the request body for self-service account deletion.
type DeleteAccountRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
}
