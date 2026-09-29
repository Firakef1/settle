package dto

// ResendCodeRequest is the request body for POST /auth/resend-verification.
type ResendCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}
