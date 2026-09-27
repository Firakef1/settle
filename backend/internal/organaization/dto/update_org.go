package dto

type UpdateOrgRequest struct {
	Name     *string `json:"name,omitempty"`
	Currency *string `json:"currency,omitempty"`
}
