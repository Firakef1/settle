package dto

// ForgotPasswordRequest is the request body for requesting a password reset OTP.
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}
