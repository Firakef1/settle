package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Firakef1/settle/backend/internal/organaization/dto"
	"github.com/Firakef1/settle/backend/internal/organaization/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =========================================================================
// Domain 5: Audit Log Tests (`GET /organizations/:id/audit-log`)
// =========================================================================

func TestGetAuditLog_OrgAdmin_Success(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-audit-1"
	adminID := "admin-audit-1"

	env.orgRepo.Orgs[orgID] = &model.Organization{
		ID:        orgID,
		Name:      "Audit Test Org",
		Slug:      "audit-test-org",
		Currency:  "USD",
		Plan:      "free",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	env.memberRepo.Members[orgID+":"+adminID] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   adminID,
		Role:     "org_admin",
		JoinedAt: time.Now(),
	}

	// Add sample audit logs
	metaBytes, _ := json.Marshal(map[string]any{"note": "org setup"})
	env.auditWriter.Logs = append(env.auditWriter.Logs,
		&model.AuditLog{
			ID:        "log-1",
			OrgID:     orgID,
			ActorID:   adminID,
			Action:    "org_created",
			Metadata:  string(metaBytes),
			CreatedAt: time.Now().Add(-2 * time.Hour),
		},
		&model.AuditLog{
			ID:        "log-2",
			OrgID:     orgID,
			ActorID:   adminID,
			Action:    "member_invited",
			Metadata:  string(metaBytes),
			CreatedAt: time.Now().Add(-1 * time.Hour),
		},
	)

	token := generateTestJWT(adminID, orgID, "org_admin")
	req, _ := http.NewRequest(http.MethodGet, "/v1/organizations/"+orgID+"/audit-log", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.AuditLogListResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, 2, resp.Pagination.Total)
	assert.Equal(t, 2, len(resp.Data))
	assert.Equal(t, "org_created", resp.Data[0].Action)
}

func TestGetAuditLog_Finance_Success(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-audit-2"
	financeID := "finance-audit-1"

	env.orgRepo.Orgs[orgID] = &model.Organization{
		ID:        orgID,
		Name:      "Finance Audit Org",
		Slug:      "finance-audit-org",
		Currency:  "USD",
		Plan:      "free",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	env.memberRepo.Members[orgID+":"+financeID] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   financeID,
		Role:     "finance",
		JoinedAt: time.Now(),
	}

	token := generateTestJWT(financeID, orgID, "finance")
	req, _ := http.NewRequest(http.MethodGet, "/v1/organizations/"+orgID+"/audit-log", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetAuditLog_FilterByAction(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-audit-3"
	adminID := "admin-audit-3"

	env.memberRepo.Members[orgID+":"+adminID] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   adminID,
		Role:     "org_admin",
		JoinedAt: time.Now(),
	}

	env.auditWriter.Logs = append(env.auditWriter.Logs,
		&model.AuditLog{
			ID:        "log-a",
			OrgID:     orgID,
			ActorID:   adminID,
			Action:    "role_updated",
			CreatedAt: time.Now().Add(-2 * time.Hour),
		},
		&model.AuditLog{
			ID:        "log-b",
			OrgID:     orgID,
			ActorID:   adminID,
			Action:    "plan_updated",
			CreatedAt: time.Now().Add(-1 * time.Hour),
		},
	)

	token := generateTestJWT(adminID, orgID, "org_admin")
	req, _ := http.NewRequest(http.MethodGet, "/v1/organizations/"+orgID+"/audit-log?action=plan_updated", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.AuditLogListResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, 1, resp.Pagination.Total)
	assert.Equal(t, 1, len(resp.Data))
	assert.Equal(t, "plan_updated", resp.Data[0].Action)
}

func TestGetAuditLog_Pagination(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-audit-4"
	adminID := "admin-audit-4"

	env.memberRepo.Members[orgID+":"+adminID] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   adminID,
		Role:     "org_admin",
		JoinedAt: time.Now(),
	}

	for i := 1; i <= 5; i++ {
		env.auditWriter.Logs = append(env.auditWriter.Logs, &model.AuditLog{
			ID:        fmt.Sprintf("log-%d", i),
			OrgID:     orgID,
			ActorID:   adminID,
			Action:    "action_test",
			CreatedAt: time.Now(),
		})
	}

	token := generateTestJWT(adminID, orgID, "org_admin")
	req, _ := http.NewRequest(http.MethodGet, "/v1/organizations/"+orgID+"/audit-log?limit=2&offset=2", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp dto.AuditLogListResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, 5, resp.Pagination.Total)
	assert.Equal(t, 2, len(resp.Data))
	assert.True(t, resp.Pagination.HasMore)
}

func TestGetAuditLog_StaffRole_Returns403(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-audit-5"
	staffID := "staff-audit-1"

	token := generateTestJWT(staffID, orgID, "staff")
	req, _ := http.NewRequest(http.MethodGet, "/v1/organizations/"+orgID+"/audit-log", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetAuditLog_NonMember_Returns403(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-audit-6"
	outsiderID := "outsider-1"

	token := generateTestJWT(outsiderID, orgID, "org_admin")
	req, _ := http.NewRequest(http.MethodGet, "/v1/organizations/"+orgID+"/audit-log", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "forbidden: user is not a member of this organization")
}

func TestGetAuditLog_Unauthenticated_Returns401(t *testing.T) {
	env := setupTestEnv()
	req, _ := http.NewRequest(http.MethodGet, "/v1/organizations/org-1/audit-log", nil)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// =========================================================================
// Domain 4: Organization Stats Tests (`GET /organizations/:id/stats`)
// =========================================================================

func TestGetOrgStats_OrgAdmin_Success(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-stats-1"
	adminID := "admin-stats-1"

	env.orgRepo.Orgs[orgID] = &model.Organization{
		ID:        orgID,
		Name:      "Metrics Test Org",
		Slug:      "metrics-test-org",
		Currency:  "USD",
		Plan:      "free",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	env.memberRepo.Members[orgID+":"+adminID] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   adminID,
		Role:     "org_admin",
		JoinedAt: time.Now(),
	}

	env.statsRepo.MemberRoleCounts[orgID] = map[string]int{
		"org_admin": 1,
		"finance":   2,
		"staff":     7,
	}
	env.statsRepo.RequestStatusCounts[orgID] = map[string]int{
		"pending":   4,
		"approved":  3,
		"paid":      10,
		"rejected":  2,
		"failed":    1,
		"withdrawn": 1,
	}
	env.statsRepo.TotalSpent[orgID] = 4500.50
	env.statsRepo.ThisMonthSpent[orgID] = 1200.00
	env.statsRepo.RequestsThisMonth[orgID] = 15

	token := generateTestJWT(adminID, orgID, "org_admin")
	req, _ := http.NewRequest(http.MethodGet, "/v1/organizations/"+orgID+"/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]dto.OrgStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	stats := resp["data"]
	assert.Equal(t, orgID, stats.OrgID)

	// Member counts
	assert.Equal(t, 10, stats.MemberCounts.Total)
	assert.Equal(t, 1, stats.MemberCounts.OrgAdmin)
	assert.Equal(t, 2, stats.MemberCounts.Finance)
	assert.Equal(t, 7, stats.MemberCounts.Staff)

	// Request stats
	assert.Equal(t, 21, stats.RequestStats.Total)
	assert.Equal(t, 4, stats.RequestStats.Pending)
	assert.Equal(t, 3, stats.RequestStats.Approved)
	assert.Equal(t, 10, stats.RequestStats.Paid)

	// Financials
	assert.Equal(t, 4500.50, stats.Financials.TotalSpent)
	assert.Equal(t, 1200.00, stats.Financials.ThisMonthSpent)
	assert.Equal(t, "USD", stats.Financials.Currency)

	// Plan usage (Free tier limits)
	assert.Equal(t, "free", stats.PlanUsage.CurrentPlan)
	assert.Equal(t, 100, stats.PlanUsage.RequestLimit)
	assert.Equal(t, 5, stats.PlanUsage.UserLimit)
	assert.Equal(t, 10, stats.PlanUsage.UserCount)
	assert.Equal(t, 15, stats.PlanUsage.RequestsThisMonth)
}

func TestGetOrgStats_FinanceAccess_Success(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-stats-2"
	financeID := "finance-stats-1"

	env.orgRepo.Orgs[orgID] = &model.Organization{
		ID:        orgID,
		Name:      "Finance Stats Org",
		Slug:      "finance-stats-org",
		Currency:  "EUR",
		Plan:      "starter",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	env.memberRepo.Members[orgID+":"+financeID] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   financeID,
		Role:     "finance",
		JoinedAt: time.Now(),
	}

	token := generateTestJWT(financeID, orgID, "finance")
	req, _ := http.NewRequest(http.MethodGet, "/v1/organizations/"+orgID+"/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]dto.OrgStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	stats := resp["data"]
	assert.Equal(t, "starter", stats.PlanUsage.CurrentPlan)
	assert.Equal(t, 1000, stats.PlanUsage.RequestLimit)
	assert.Equal(t, 25, stats.PlanUsage.UserLimit)
}

func TestGetOrgStats_StaffRole_Returns403(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-stats-3"
	staffID := "staff-1"

	token := generateTestJWT(staffID, orgID, "staff")
	req, _ := http.NewRequest(http.MethodGet, "/v1/organizations/"+orgID+"/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetOrgStats_NonMember_Returns403(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-stats-4"
	outsiderID := "user-outsider"

	env.orgRepo.Orgs[orgID] = &model.Organization{
		ID:        orgID,
		Name:      "Stats Org",
		Slug:      "stats-org",
		Currency:  "USD",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	token := generateTestJWT(outsiderID, orgID, "org_admin")
	req, _ := http.NewRequest(http.MethodGet, "/v1/organizations/"+orgID+"/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetOrgStats_OrgNotFound_Returns404(t *testing.T) {
	env := setupTestEnv()
	orgID := "non-existent-org"
	adminID := "admin-1"

	env.memberRepo.Members[orgID+":"+adminID] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   adminID,
		Role:     "org_admin",
		JoinedAt: time.Now(),
	}

	token := generateTestJWT(adminID, orgID, "org_admin")
	req, _ := http.NewRequest(http.MethodGet, "/v1/organizations/"+orgID+"/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// =========================================================================
// Domain 6: Plan & Billing Tests (`PUT /organizations/:id/plan` & `GET /billing/plans`)
// =========================================================================

func TestUpdatePlan_Success(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-plan-1"
	adminID := "admin-plan-1"

	env.orgRepo.Orgs[orgID] = &model.Organization{
		ID:        orgID,
		Name:      "Billing Org",
		Slug:      "billing-org",
		Currency:  "USD",
		Plan:      "free",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	env.memberRepo.Members[orgID+":"+adminID] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   adminID,
		Role:     "org_admin",
		JoinedAt: time.Now(),
	}

	token := generateTestJWT(adminID, orgID, "org_admin")
	body := dto.UpdatePlanRequest{Plan: "starter"}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/v1/organizations/"+orgID+"/plan", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]dto.UpdatePlanResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "starter", resp["data"].Plan)
	assert.Equal(t, "starter", env.orgRepo.Orgs[orgID].Plan)
	assert.True(t, env.auditWriter.HasAction("plan_updated"))
}

func TestUpdatePlan_InvalidPlan(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-plan-2"
	adminID := "admin-plan-2"

	env.orgRepo.Orgs[orgID] = &model.Organization{
		ID:        orgID,
		Name:      "Billing Org",
		Slug:      "billing-org",
		Currency:  "USD",
		Plan:      "free",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	env.memberRepo.Members[orgID+":"+adminID] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   adminID,
		Role:     "org_admin",
		JoinedAt: time.Now(),
	}

	token := generateTestJWT(adminID, orgID, "org_admin")
	body := dto.UpdatePlanRequest{Plan: "enterprise"} // invalid
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/v1/organizations/"+orgID+"/plan", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid plan")
}

func TestUpdatePlan_ForbiddenForFinanceAndStaff(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-plan-3"

	body := dto.UpdatePlanRequest{Plan: "starter"}
	jsonBody, _ := json.Marshal(body)

	// Finance role attempt
	financeToken := generateTestJWT("user-finance", orgID, "finance")
	req, _ := http.NewRequest(http.MethodPut, "/v1/organizations/"+orgID+"/plan", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+financeToken)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	// Staff role attempt
	staffToken := generateTestJWT("user-staff", orgID, "staff")
	req2, _ := http.NewRequest(http.MethodPut, "/v1/organizations/"+orgID+"/plan", bytes.NewBuffer(jsonBody))
	req2.Header.Set("Authorization", "Bearer "+staffToken)
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	env.router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusForbidden, w2.Code)
}

func TestGetBillingPlans_Public_Success(t *testing.T) {
	env := setupTestEnv()

	// No Authorization header required (public endpoint)
	req, _ := http.NewRequest(http.MethodGet, "/v1/billing/plans", nil)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string][]dto.Plan
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	plans := resp["data"]
	require.Equal(t, 3, len(plans))

	// Verify plans shape
	assert.Equal(t, "free", plans[0].ID)
	assert.Equal(t, float64(0), plans[0].Price)
	assert.Equal(t, 100, plans[0].RequestLimit)
	assert.Equal(t, 5, plans[0].UserLimit)

	assert.Equal(t, "starter", plans[1].ID)
	assert.Equal(t, float64(49), plans[1].Price)
	assert.Equal(t, 1000, plans[1].RequestLimit)
	assert.Equal(t, 25, plans[1].UserLimit)

	assert.Equal(t, "pro", plans[2].ID)
	assert.Equal(t, float64(199), plans[2].Price)
	assert.Equal(t, 0, plans[2].RequestLimit) // 0 = unlimited
	assert.Equal(t, 0, plans[2].UserLimit)
}

// =========================================================================
// Domain 4: Dashboard Summary Tests (`GET /dashboard/summary`)
// =========================================================================

func TestDashboardSummary_OrgAdmin_Success(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-dash-1"
	adminID := "admin-dash-1"

	env.memberRepo.Members[orgID+":"+adminID] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   adminID,
		Role:     "org_admin",
		JoinedAt: time.Now(),
	}

	token := generateTestJWT(adminID, orgID, "org_admin")
	req, _ := http.NewRequest(http.MethodGet, "/v1/dashboard/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]dto.DashboardSummaryResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	summary := resp["data"]
	assert.Equal(t, 4, summary.PendingCount)
	assert.Equal(t, 2, summary.UrgentCount)
	assert.Equal(t, 1, summary.CriticalCount)
	assert.Equal(t, 1, summary.UrgencyBreakdown.Routine)
	assert.Equal(t, 2, summary.AgingBreakdown.ZeroToThreeDays)
	assert.Equal(t, 1, summary.AgingBreakdown.ThreeToSevenDays)
	assert.Equal(t, 1, summary.AgingBreakdown.SevenPlusDays)
	assert.Equal(t, 1, len(summary.EscalatedItems))
	assert.Equal(t, "Alice", summary.EscalatedItems[0].RequesterName)
	assert.Equal(t, 8, summary.EscalatedItems[0].DaysPending)
}

func TestDashboardSummary_FinanceAccess_Success(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-dash-2"
	financeID := "finance-dash-1"

	env.memberRepo.Members[orgID+":"+financeID] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   financeID,
		Role:     "finance",
		JoinedAt: time.Now(),
	}

	token := generateTestJWT(financeID, orgID, "finance")
	req, _ := http.NewRequest(http.MethodGet, "/v1/dashboard/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardSummary_OrgIDViaQueryParam(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-dash-3"
	adminID := "admin-dash-3"

	env.memberRepo.Members[orgID+":"+adminID] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   adminID,
		Role:     "org_admin",
		JoinedAt: time.Now(),
	}

	// JWT with empty orgID, passed via query param
	token := generateTestJWT(adminID, "", "org_admin")
	req, _ := http.NewRequest(http.MethodGet, "/v1/dashboard/summary?org_id="+orgID, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardSummary_OrgIDViaHeader(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-dash-4"
	adminID := "admin-dash-4"

	env.memberRepo.Members[orgID+":"+adminID] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   adminID,
		Role:     "org_admin",
		JoinedAt: time.Now(),
	}

	token := generateTestJWT(adminID, "", "org_admin")
	req, _ := http.NewRequest(http.MethodGet, "/v1/dashboard/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Organization-Id", orgID)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashboardSummary_MissingOrgContext_Returns400(t *testing.T) {
	env := setupTestEnv()
	adminID := "admin-no-org"

	token := generateTestJWT(adminID, "", "org_admin")
	req, _ := http.NewRequest(http.MethodGet, "/v1/dashboard/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "organization context is required")
}

func TestDashboardSummary_StaffRole_Returns403(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-dash-5"
	staffID := "staff-1"

	token := generateTestJWT(staffID, orgID, "staff")
	req, _ := http.NewRequest(http.MethodGet, "/v1/dashboard/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestDashboardSummary_Unauthenticated_Returns401(t *testing.T) {
	env := setupTestEnv()
	req, _ := http.NewRequest(http.MethodGet, "/v1/dashboard/summary", nil)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
