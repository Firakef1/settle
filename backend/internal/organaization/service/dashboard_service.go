package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Firakef1/settle/backend/internal/organaization/dto"
	"github.com/Firakef1/settle/backend/internal/organaization/repository"
)

// DashboardService handles operations for organization statistics and high-level dashboard summaries.
type DashboardService interface {
	GetOrgStats(ctx context.Context, orgID string, callerUserID string) (*dto.OrgStatsResponse, error)
	GetSummary(ctx context.Context, orgID string, callerUserID string) (*dto.DashboardSummaryResponse, error)
}

type dashboardService struct {
	db         *sql.DB
	orgRepo    repository.OrgRepository
	memberRepo repository.MemberRepository
	statsRepo  repository.StatsRepository
}

// NewDashboardService creates a new DashboardService instance.
func NewDashboardService(
	db *sql.DB,
	orgRepo repository.OrgRepository,
	memberRepo repository.MemberRepository,
	statsRepo repository.StatsRepository,
) DashboardService {
	return &dashboardService{
		db:         db,
		orgRepo:    orgRepo,
		memberRepo: memberRepo,
		statsRepo:  statsRepo,
	}
}

func (s *dashboardService) getExecutor() repository.DBTX {
	if s.db != nil {
		return s.db
	}
	return nil
}

func (s *dashboardService) GetOrgStats(ctx context.Context, orgID string, callerUserID string) (*dto.OrgStatsResponse, error) {
	exec := s.getExecutor()

	// Verify membership
	if callerUserID != "" && s.memberRepo != nil {
		_, err := s.memberRepo.GetMember(ctx, exec, orgID, callerUserID)
		if err != nil {
			if errors.Is(err, repository.ErrMemberNotFound) {
				return nil, ErrNotOrgMember
			}
			return nil, err
		}
	}

	org, err := s.orgRepo.GetByID(ctx, exec, orgID)
	if err != nil {
		return nil, err
	}

	memberCountsMap, err := s.statsRepo.GetMemberRoleCounts(ctx, exec, orgID)
	if err != nil {
		return nil, err
	}

	adminCount := memberCountsMap["org_admin"]
	financeCount := memberCountsMap["finance"]
	staffCount := memberCountsMap["staff"]
	totalMembers := adminCount + financeCount + staffCount

	requestCountsMap, err := s.statsRepo.GetRequestStatusCounts(ctx, exec, orgID)
	if err != nil {
		return nil, err
	}

	pendingCount := requestCountsMap["pending"]
	approvedCount := requestCountsMap["approved"]
	paidCount := requestCountsMap["paid"]
	rejectedCount := requestCountsMap["rejected"]
	failedCount := requestCountsMap["failed"]
	withdrawnCount := requestCountsMap["withdrawn"]
	totalRequests := pendingCount + approvedCount + paidCount + rejectedCount + failedCount + withdrawnCount

	totalSpent, thisMonthSpent, err := s.statsRepo.GetFinancialSpending(ctx, exec, orgID)
	if err != nil {
		return nil, err
	}

	requestsThisMonth, err := s.statsRepo.GetRequestsCountThisMonth(ctx, exec, orgID)
	if err != nil {
		return nil, err
	}

	plan := org.Plan
	if plan == "" {
		plan = "free"
	}

	var reqLimit, userLimit int
	switch plan {
	case "free":
		reqLimit = 100
		userLimit = 5
	case "starter":
		reqLimit = 1000
		userLimit = 25
	case "pro":
		reqLimit = 0
		userLimit = 0
	default:
		reqLimit = 100
		userLimit = 5
	}

	return &dto.OrgStatsResponse{
		OrgID: orgID,
		MemberCounts: dto.MemberCounts{
			Total:    totalMembers,
			OrgAdmin: adminCount,
			Finance:  financeCount,
			Staff:    staffCount,
		},
		RequestStats: dto.RequestStats{
			Total:     totalRequests,
			Pending:   pendingCount,
			Approved:  approvedCount,
			Paid:      paidCount,
			Rejected:  rejectedCount,
			Failed:    failedCount,
			Withdrawn: withdrawnCount,
		},
		Financials: dto.FinancialStats{
			TotalSpent:     totalSpent,
			ThisMonthSpent: thisMonthSpent,
			Currency:       org.Currency,
		},
		PlanUsage: dto.PlanUsageStats{
			CurrentPlan:       plan,
			RequestsThisMonth: requestsThisMonth,
			RequestLimit:      reqLimit,
			UserCount:         totalMembers,
			UserLimit:         userLimit,
		},
	}, nil
}

func (s *dashboardService) GetSummary(ctx context.Context, orgID string, callerUserID string) (*dto.DashboardSummaryResponse, error) {
	exec := s.getExecutor()

	// Verify caller belongs to organization if provided
	if callerUserID != "" && s.memberRepo != nil {
		_, err := s.memberRepo.GetMember(ctx, exec, orgID, callerUserID)
		if err != nil {
			if errors.Is(err, repository.ErrMemberNotFound) {
				return nil, ErrNotOrgMember
			}
			return nil, err
		}
	}

	return s.statsRepo.GetDashboardSummary(ctx, exec, orgID)
}
