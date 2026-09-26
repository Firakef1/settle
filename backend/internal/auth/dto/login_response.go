package dto

import "github.com/Firakef1/settle/backend/internal/auth/model"

// UserResponseDTO represents the user information returned in login/signup response.
type UserResponseDTO struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// OrgMembershipDTO represents membership info returned in login response.
type OrgMembershipDTO struct {
	OrgID      string `json:"org_id"`
	OrgName    string `json:"org_name"`
	OrgSlug    string `json:"org_slug"`
	Role       string `json:"role"`
	Department string `json:"department,omitempty"`
}

// LoginResponse represents the response payload for successful login.
type LoginResponse struct {
	Token        string             `json:"token"`
	RefreshToken string             `json:"refresh_token"`
	User         UserResponseDTO    `json:"user"`
	Orgs         []OrgMembershipDTO `json:"orgs"`
}

// MapUserToDTO converts user model to response DTO.
func MapUserToDTO(user *model.User) UserResponseDTO {
	return UserResponseDTO{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// MapMembershipsToDTO converts model memberships to DTO list.
func MapMembershipsToDTO(memberships []model.OrgMembership) []OrgMembershipDTO {
	dtos := make([]OrgMembershipDTO, 0, len(memberships))
	for _, m := range memberships {
		dtos = append(dtos, OrgMembershipDTO{
			OrgID:      m.OrgID,
			OrgName:    m.OrgName,
			OrgSlug:    m.OrgSlug,
			Role:       m.Role,
			Department: m.Department,
		})
	}
	return dtos
}
