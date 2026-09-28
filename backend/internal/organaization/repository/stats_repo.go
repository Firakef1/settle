package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Firakef1/settle/backend/internal/organaization/dto"
)

// StatsRepository defines database query operations for metrics and dashboard aggregates.
type StatsRepository interface {
	GetMemberRoleCounts(ctx context.Context, db DBTX, orgID string) (map[string]int, error)
	GetRequestStatusCounts(ctx context.Context, db DBTX, orgID string) (map[string]int, error)
	GetFinancialSpending(ctx context.Context, db DBTX, orgID string) (totalSpent float64, thisMonthSpent float64, err error)
	GetRequestsCountThisMonth(ctx context.Context, db DBTX, orgID string) (int, error)
	GetDashboardSummary(ctx context.Context, db DBTX, orgID string) (*dto.DashboardSummaryResponse, error)
}

type pgStatsRepo struct{}

// NewStatsRepository creates a new PostgreSQL implementation of StatsRepository.
func NewStatsRepository() StatsRepository {
	return &pgStatsRepo{}
}

func (r *pgStatsRepo) GetMemberRoleCounts(ctx context.Context, db DBTX, orgID string) (map[string]int, error) {
	query := `
		SELECT role, COUNT(*)
		FROM org_members
		WHERE org_id = $1 AND status != 'removed'
		GROUP BY role
	`
	rows, err := db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query member role counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var role string
		var count int
		if err := rows.Scan(&role, &count); err != nil {
			return nil, fmt.Errorf("failed to scan member role count: %w", err)
		}
		counts[role] = count
	}

	return counts, rows.Err()
}

func (r *pgStatsRepo) GetRequestStatusCounts(ctx context.Context, db DBTX, orgID string) (map[string]int, error) {
	query := `
		SELECT status, COUNT(*)
		FROM requests
		WHERE org_id = $1
		GROUP BY status
	`
	rows, err := db.QueryContext(ctx, query, orgID)
	if err != nil {
		// If requests table does not exist yet in early development, return empty counts gracefully
		if strings.Contains(err.Error(), "relation \"requests\" does not exist") {
			return make(map[string]int), nil
		}
		return nil, fmt.Errorf("failed to query request status counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan request status count: %w", err)
		}
		counts[status] = count
	}

	return counts, rows.Err()
}

func (r *pgStatsRepo) GetFinancialSpending(ctx context.Context, db DBTX, orgID string) (float64, float64, error) {
	query := `
		SELECT
			COALESCE(SUM(amount), 0),
			COALESCE(SUM(CASE WHEN created_at >= date_trunc('month', NOW()) THEN amount ELSE 0 END), 0)
		FROM requests
		WHERE org_id = $1 AND status = 'paid'
	`
	var totalSpent, thisMonthSpent float64
	err := db.QueryRowContext(ctx, query, orgID).Scan(&totalSpent, &thisMonthSpent)
	if err != nil {
		if strings.Contains(err.Error(), "relation \"requests\" does not exist") {
			return 0, 0, nil
		}
		return 0, 0, fmt.Errorf("failed to query financial spending: %w", err)
	}
	return totalSpent, thisMonthSpent, nil
}

func (r *pgStatsRepo) GetRequestsCountThisMonth(ctx context.Context, db DBTX, orgID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM requests
		WHERE org_id = $1 AND created_at >= date_trunc('month', NOW())
	`
	var count int
	err := db.QueryRowContext(ctx, query, orgID).Scan(&count)
	if err != nil {
		if strings.Contains(err.Error(), "relation \"requests\" does not exist") {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to query requests count this month: %w", err)
	}
	return count, nil
}

func (r *pgStatsRepo) GetDashboardSummary(ctx context.Context, db DBTX, orgID string) (*dto.DashboardSummaryResponse, error) {
	summaryQuery := `
		SELECT
			COUNT(*) as pending_count,
			COUNT(*) FILTER (WHERE urgency = 'urgent') as urgent_count,
			COUNT(*) FILTER (WHERE urgency = 'critical') as critical_count,
			COUNT(*) FILTER (WHERE urgency = 'routine') as routine_count,
			COUNT(*) FILTER (WHERE EXTRACT(DAY FROM NOW() - created_at) <= 3) as age_0_3,
			COUNT(*) FILTER (WHERE EXTRACT(DAY FROM NOW() - created_at) > 3 AND EXTRACT(DAY FROM NOW() - created_at) <= 7) as age_3_7,
			COUNT(*) FILTER (WHERE EXTRACT(DAY FROM NOW() - created_at) > 7) as age_7_plus
		FROM requests
		WHERE org_id = $1 AND status = 'pending'
	`
	resp := &dto.DashboardSummaryResponse{
		EscalatedItems: make([]dto.EscalatedItem, 0),
	}

	var routineCount, age03, age37, age7Plus int
	err := db.QueryRowContext(ctx, summaryQuery, orgID).Scan(
		&resp.PendingCount,
		&resp.UrgentCount,
		&resp.CriticalCount,
		&routineCount,
		&age03,
		&age37,
		&age7Plus,
	)
	if err != nil {
		if strings.Contains(err.Error(), "relation \"requests\" does not exist") {
			return resp, nil
		}
		return nil, fmt.Errorf("failed to query dashboard summary: %w", err)
	}

	resp.UrgencyBreakdown = dto.UrgencyBreakdown{
		Routine:  routineCount,
		Urgent:   resp.UrgentCount,
		Critical: resp.CriticalCount,
	}

	resp.AgingBreakdown = dto.AgingBreakdown{
		ZeroToThreeDays:  age03,
		ThreeToSevenDays: age37,
		SevenPlusDays:    age7Plus,
	}

	escalatedQuery := `
		SELECT r.id, r.requester_id, COALESCE(u.name, ''), r.amount, r.urgency,
		       COALESCE(EXTRACT(DAY FROM NOW() - r.created_at)::int, 0) as days_pending,
		       r.created_at
		FROM requests r
		LEFT JOIN users u ON r.requester_id = u.id
		WHERE r.org_id = $1 AND r.status = 'pending' AND NOW() - r.created_at > INTERVAL '7 days'
		ORDER BY r.created_at ASC
		LIMIT 50
	`
	rows, err := db.QueryContext(ctx, escalatedQuery, orgID)
	if err != nil {
		if strings.Contains(err.Error(), "relation \"requests\" does not exist") {
			return resp, nil
		}
		return nil, fmt.Errorf("failed to query escalated items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item dto.EscalatedItem
		var reqName sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.RequesterID,
			&reqName,
			&item.Amount,
			&item.Urgency,
			&item.DaysPending,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan escalated item: %w", err)
		}
		if reqName.Valid {
			item.RequesterName = reqName.String
		}
		resp.EscalatedItems = append(resp.EscalatedItems, item)
	}

	return resp, rows.Err()
}
