package service

import (
	"context"

	"github.com/Firakef1/settle/backend/internal/organaization/dto"
)

// BillingService manages subscription plans and billing operations.
type BillingService interface {
	GetPlans() []dto.Plan
	UpdatePlan(ctx context.Context, orgID string, actorID string, newPlan string) (*dto.UpdatePlanResponse, error)
}

type billingService struct {
	orgService OrgService
}

// NewBillingService constructs a new BillingService.
func NewBillingService(orgService OrgService) BillingService {
	return &billingService{
		orgService: orgService,
	}
}

// GetPlans returns the platform's available subscription tiers.
func (s *billingService) GetPlans() []dto.Plan {
	return []dto.Plan{
		{
			ID:           "free",
			Name:         "Free",
			Price:        0,
			RequestLimit: 100,
			UserLimit:    5,
			Features: []string{
				"Up to 5 team members",
				"100 payout requests / month",
				"Standard audit logs",
				"Basic email support",
			},
		},
		{
			ID:           "starter",
			Name:         "Starter",
			Price:        49,
			RequestLimit: 1000,
			UserLimit:    25,
			Features: []string{
				"Up to 25 team members",
				"1,000 payout requests / month",
				"Advanced audit log search & export",
				"Priority email support",
				"Automated receipt OCR extraction",
			},
		},
		{
			ID:           "pro",
			Name:         "Pro",
			Price:        199,
			RequestLimit: 0, // unlimited
			UserLimit:    0, // unlimited
			Features: []string{
				"Unlimited team members",
				"Unlimited payout requests",
				"Full compliance audit exports",
				"Custom approval workflows",
				"24/7 dedicated support",
			},
		},
	}
}

func (s *billingService) UpdatePlan(ctx context.Context, orgID string, actorID string, newPlan string) (*dto.UpdatePlanResponse, error) {
	orgResp, err := s.orgService.UpdatePlan(ctx, orgID, actorID, newPlan)
	if err != nil {
		return nil, err
	}

	return &dto.UpdatePlanResponse{
		OrgID:     orgResp.ID,
		Plan:      orgResp.Plan,
		UpdatedAt: orgResp.UpdatedAt,
	}, nil
}
