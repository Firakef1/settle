package dto

// VerifyEmailRequest is the request body for verifying an email address.
type VerifyEmailRequest struct {
	Email            string `json:"email" binding:"required,email"`
	Code             string `json:"code"`
	VerificationCode string `json:"verification_code"`
}

// GetCode returns whichever code field was provided in JSON.
func (r *VerifyEmailRequest) GetCode() string {
	if r.Code != "" {
		return r.Code
	}
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
