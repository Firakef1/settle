package dto

// ApproveRequest is the optional body for approving a pending request.
type ApproveRequest struct {
	Note string `json:"note"`
}
