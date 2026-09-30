package dto

// VerifyEmailRequest is the request body for verifying an email address.
type VerifyEmailRequest struct {
	Email            string `json:"email" binding:"required,email"`
	VerificationCode string `json:"verification_code" binding:"required"`
}

// GetCode returns the verification code.
func (r *VerifyEmailRequest) GetCode() string {
	return r.VerificationCode
}

// VerifyEmailResponse is the response body returned after verifying email.
type VerifyEmailResponse struct {
	Message      string             `json:"message"`
	Token        string             `json:"token,omitempty"`
	RefreshToken string             `json:"refresh_token,omitempty"`
	User         *UserResponseDTO   `json:"user,omitempty"`
	Orgs         []OrgMembershipDTO `json:"orgs,omitempty"`
}
