package dto

// RejectRequest rejects a pending request. Reason is required.
type RejectRequest struct {
	Reason string `json:"reason"`
}
