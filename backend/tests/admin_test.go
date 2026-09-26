package tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Firakef1/settle/backend/internal/organaization/dto"
	orghandler "github.com/Firakef1/settle/backend/internal/organaization/handler"
	"github.com/Firakef1/settle/backend/internal/organaization/model"
	"github.com/Firakef1/settle/backend/internal/organaization/repository"
	"github.com/Firakef1/settle/backend/internal/organaization/service"
	"github.com/Firakef1/settle/backend/internal/router"
	"github.com/Firakef1/settle/backend/internal/shared/middleware"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
	_ = os.Setenv("JWT_SECRET", "test-secret")
}

// --- Mocks ---

type MockTxRunner struct {
	FailWithTx bool
}

func (m *MockTxRunner) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	if m.FailWithTx {
		return errors.New("simulated transaction failure")
	}
	return fn(nil)
}

type MockOrgRepo struct {
	mu           sync.Mutex
	Orgs         map[string]*model.Organization
	FailCreateTx bool
	FailGet      bool
	FailUpdate   bool
}

func NewMockOrgRepo() *MockOrgRepo {
	return &MockOrgRepo{
		Orgs: make(map[string]*model.Organization),
	}
}

func (m *MockOrgRepo) CreateTx(ctx context.Context, tx *sql.Tx, org *model.Organization) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FailCreateTx {
		return errors.New("failed to insert org in tx")
	}
	m.Orgs[org.ID] = org
	return nil
}

func (m *MockOrgRepo) GetByID(ctx context.Context, db repository.DBTX, id string) (*model.Organization, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FailGet {
		return nil, errors.New("db error")
	}
	org, ok := m.Orgs[id]
	if !ok {
		return nil, repository.ErrOrgNotFound
	}
	return org, nil
}

func (m *MockOrgRepo) GetBySlug(ctx context.Context, db repository.DBTX, slug string) (*model.Organization, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, o := range m.Orgs {
		if o.Slug == slug {
			return o, nil
		}
	}
	return nil, repository.ErrOrgNotFound
}

func (m *MockOrgRepo) Update(ctx context.Context, db repository.DBTX, org *model.Organization) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FailUpdate {
		return errors.New("failed to update org")
	}
	if _, ok := m.Orgs[org.ID]; !ok {
		return repository.ErrOrgNotFound
	}
	m.Orgs[org.ID] = org
	return nil
}

type MockMemberRepo struct {
	mu         sync.Mutex
	Members    map[string]*model.OrgMember // key: orgID + ":" + userID
	UserNames  map[string]string
	UserEmails map[string]string
	FailAddTx  bool
	FailDelete bool
	FailUpdate bool
}

func NewMockMemberRepo() *MockMemberRepo {
	return &MockMemberRepo{
		Members:    make(map[string]*model.OrgMember),
		UserNames:  make(map[string]string),
		UserEmails: make(map[string]string),
	}
}

func (m *MockMemberRepo) AddTx(ctx context.Context, tx *sql.Tx, member *model.OrgMember) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FailAddTx {
		return errors.New("failed to add member in tx")
	}
	key := member.OrgID + ":" + member.UserID
	m.Members[key] = member
	return nil
}

func (m *MockMemberRepo) Add(ctx context.Context, db repository.DBTX, member *model.OrgMember) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := member.OrgID + ":" + member.UserID
	m.Members[key] = member
	return nil
}

func (m *MockMemberRepo) GetMember(ctx context.Context, db repository.DBTX, orgID, userID string) (*model.OrgMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := orgID + ":" + userID
	member, ok := m.Members[key]
	if !ok {
		return nil, repository.ErrMemberNotFound
	}
	return member, nil
}

func (m *MockMemberRepo) ListByOrgID(ctx context.Context, db repository.DBTX, orgID string) ([]*dto.MemberResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []*dto.MemberResponse
	for _, member := range m.Members {
		if member.OrgID == orgID {
			list = append(list, &dto.MemberResponse{
				UserID:   member.UserID,
				Name:     m.UserNames[member.UserID],
				Email:    m.UserEmails[member.UserID],
				Role:     member.Role,
				JoinedAt: member.JoinedAt,
			})
		}
	}
	return list, nil
}

func (m *MockMemberRepo) CountAdmins(ctx context.Context, db repository.DBTX, orgID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, member := range m.Members {
		if member.OrgID == orgID && member.Role == "org_admin" {
			count++
		}
	}
	return count, nil
}

func (m *MockMemberRepo) Delete(ctx context.Context, db repository.DBTX, orgID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FailDelete {
		return errors.New("failed to delete member")
	}
	key := orgID + ":" + userID
	if _, ok := m.Members[key]; !ok {
		return repository.ErrMemberNotFound
	}
	delete(m.Members, key)
	return nil
}

func (m *MockMemberRepo) UpdateRole(ctx context.Context, db repository.DBTX, orgID, userID, newRole string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FailUpdate {
		return errors.New("failed to update role")
	}
	key := orgID + ":" + userID
	member, ok := m.Members[key]
	if !ok {
		return repository.ErrMemberNotFound
	}
	member.Role = newRole
	return nil
}

type MockInvitationRepo struct {
	mu          sync.Mutex
	Invitations map[string]*model.Invitation // key: token
	FailCreate  bool
	FailMark    bool
}

func NewMockInvitationRepo() *MockInvitationRepo {
	return &MockInvitationRepo{
		Invitations: make(map[string]*model.Invitation),
	}
}

func (m *MockInvitationRepo) Create(ctx context.Context, db repository.DBTX, inv *model.Invitation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FailCreate {
		return errors.New("failed to create invitation")
	}
	m.Invitations[inv.Token] = inv
	return nil
}

func (m *MockInvitationRepo) GetByToken(ctx context.Context, db repository.DBTX, token string) (*model.Invitation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	inv, ok := m.Invitations[token]
	if !ok {
		return nil, repository.ErrInvitationNotFound
	}
	return inv, nil
}

func (m *MockInvitationRepo) MarkUsedTx(ctx context.Context, tx *sql.Tx, token string, usedAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FailMark {
		return errors.New("failed to mark invitation used")
	}
	inv, ok := m.Invitations[token]
	if !ok || inv.UsedAt != nil {
		return errors.New("invitation already used or not found")
	}
	inv.UsedAt = &usedAt
	return nil
}

type MockUserRepo struct {
	mu         sync.Mutex
	Users      map[string]*model.User // key: email
	FailCreate bool
}

func NewMockUserRepo() *MockUserRepo {
	return &MockUserRepo{
		Users: make(map[string]*model.User),
	}
}

func (m *MockUserRepo) GetByEmail(ctx context.Context, db repository.DBTX, email string) (*model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.Users[strings.ToLower(email)]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	return u, nil
}

func (m *MockUserRepo) CreateTx(ctx context.Context, tx *sql.Tx, user *model.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FailCreate {
		return errors.New("failed to create user in tx")
	}
	m.Users[strings.ToLower(user.Email)] = user
	return nil
}

type MockAuditWriter struct {
	mu        sync.Mutex
	Logs      []*model.AuditLog
	FailTx    bool
	FailNonTx bool
}

func NewMockAuditWriter() *MockAuditWriter {
	return &MockAuditWriter{
		Logs: make([]*model.AuditLog, 0),
	}
}

func (m *MockAuditWriter) WriteTx(ctx context.Context, tx *sql.Tx, log *model.AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FailTx {
		return errors.New("simulated audit log tx write failure")
	}
	m.Logs = append(m.Logs, log)
	return nil
}

func (m *MockAuditWriter) Write(ctx context.Context, db repository.DBTX, log *model.AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FailNonTx {
		return errors.New("simulated audit log non-tx write failure")
	}
	m.Logs = append(m.Logs, log)
	return nil
}

func (m *MockAuditWriter) HasAction(action string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, l := range m.Logs {
		if l.Action == action {
			return true
		}
	}
	return false
}

// --- Helper to generate test JWT ---

func generateTestJWT(userID, orgID, role string) string {
	claims := &middleware.CustomClaims{
		UserID: userID,
		OrgID:  orgID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte("test-secret"))
	return tokenString
}

// --- Setup Test Engine ---

type testEnvironment struct {
	router         *gin.Engine
	orgRepo        *MockOrgRepo
	memberRepo     *MockMemberRepo
	invitationRepo *MockInvitationRepo
	userRepo       *MockUserRepo
	auditWriter    *MockAuditWriter
	txRunner       *MockTxRunner
	orgService     service.OrgService
	memberService  service.MemberService
	invService     service.InvitationService
}

func setupTestEnv() *testEnvironment {
	txRunner := &MockTxRunner{}
	orgRepo := NewMockOrgRepo()
	memberRepo := NewMockMemberRepo()
	invitationRepo := NewMockInvitationRepo()
	userRepo := NewMockUserRepo()
	auditWriter := NewMockAuditWriter()

	orgService := service.NewOrgServiceWithTx(txRunner, orgRepo, memberRepo, auditWriter)
	memberService := service.NewMemberService(nil, memberRepo, auditWriter)
	invService := service.NewInvitationServiceWithTx(txRunner, invitationRepo, memberRepo, userRepo, auditWriter)

	orgHandler := orghandler.NewOrgHandler(orgService)
	memberHandler := orghandler.NewMemberHandler(memberService)
	invHandler := orghandler.NewInvitationHandler(invService)

	r := router.SetupRouter(orgHandler, memberHandler, invHandler)

	return &testEnvironment{
		router:         r,
		orgRepo:        orgRepo,
		memberRepo:     memberRepo,
		invitationRepo: invitationRepo,
		userRepo:       userRepo,
		auditWriter:    auditWriter,
		txRunner:       txRunner,
		orgService:     orgService,
		memberService:  memberService,
		invService:     invService,
	}
}

// --- Unit & Handler Tests ---

func TestCreateOrg_Success(t *testing.T) {
	env := setupTestEnv()
	token := generateTestJWT("user-1", "", "")

	body := dto.CreateOrgRequest{
		Name:     "Acme Corp",
		Currency: "USD",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/v1/organizations", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]dto.OrgResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	org := resp["data"]
	assert.NotEmpty(t, org.ID)
	assert.Equal(t, "Acme Corp", org.Name)
	assert.Equal(t, "USD", org.Currency)
	assert.True(t, strings.HasPrefix(org.Slug, "acme-corp-"))

	// Verify creator became org_admin
	memberKey := org.ID + ":user-1"
	member, exists := env.memberRepo.Members[memberKey]
	assert.True(t, exists)
	assert.Equal(t, "org_admin", member.Role)

	// Verify audit log
	assert.True(t, env.auditWriter.HasAction("org_created"))
}

func TestCreateOrg_RollbackOnAuditFailure(t *testing.T) {
	env := setupTestEnv()
	env.auditWriter.FailTx = true // Audit write fails

	token := generateTestJWT("user-1", "", "")
	body := dto.CreateOrgRequest{
		Name:     "Rollback Corp",
		Currency: "EUR",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/v1/organizations", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateOrg_InvalidCurrency(t *testing.T) {
	env := setupTestEnv()
	token := generateTestJWT("user-1", "", "")

	body := dto.CreateOrgRequest{
		Name:     "Acme Corp",
		Currency: "JPY", // Invalid
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/v1/organizations", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid currency code")
}

func TestGetOrg_MemberAccess_Success(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-1"
	userID := "user-1"

	env.orgRepo.Orgs[orgID] = &model.Organization{
		ID:        orgID,
		Name:      "Test Org",
		Slug:      "test-org",
		Currency:  "USD",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	env.memberRepo.Members[orgID+":"+userID] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   userID,
		Role:     "staff",
		JoinedAt: time.Now(),
	}

	token := generateTestJWT(userID, orgID, "staff")

	req, _ := http.NewRequest(http.MethodGet, "/v1/organizations/"+orgID, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Test Org")
}

func TestGetOrg_NotMember_Returns403(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-1"
	nonMemberID := "user-outsider"

	env.orgRepo.Orgs[orgID] = &model.Organization{
		ID:        orgID,
		Name:      "Test Org",
		Slug:      "test-org",
		Currency:  "USD",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	token := generateTestJWT(nonMemberID, "", "")

	req, _ := http.NewRequest(http.MethodGet, "/v1/organizations/"+orgID, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUpdateOrg_InvalidCurrency(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-1"
	userID := "user-admin"

	env.orgRepo.Orgs[orgID] = &model.Organization{
		ID:        orgID,
		Name:      "Test Org",
		Slug:      "test-org",
		Currency:  "USD",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	token := generateTestJWT(userID, orgID, "org_admin")

	badCurrency := "INVALID"
	body := dto.UpdateOrgRequest{
		Currency: &badCurrency,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/v1/organizations/"+orgID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid currency code")
}

func TestUpdateOrg_Success(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-1"
	userID := "user-admin"

	env.orgRepo.Orgs[orgID] = &model.Organization{
		ID:        orgID,
		Name:      "Old Name",
		Slug:      "old-name",
		Currency:  "USD",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	token := generateTestJWT(userID, orgID, "org_admin")

	newName := "New Name"
	newCurr := "GBP"
	body := dto.UpdateOrgRequest{
		Name:     &newName,
		Currency: &newCurr,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/v1/organizations/"+orgID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "New Name", env.orgRepo.Orgs[orgID].Name)
	assert.Equal(t, "GBP", env.orgRepo.Orgs[orgID].Currency)
	assert.True(t, env.auditWriter.HasAction("org_updated"))
}

func TestListMembers_Authorized(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-1"
	token := generateTestJWT("user-admin", orgID, "org_admin")

	env.memberRepo.Members[orgID+":user-1"] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   "user-1",
		Role:     "staff",
		JoinedAt: time.Now(),
	}
	env.memberRepo.UserNames["user-1"] = "Alice"
	env.memberRepo.UserEmails["user-1"] = "alice@example.com"

	req, _ := http.NewRequest(http.MethodGet, "/v1/organizations/"+orgID+"/members", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Alice")
}

func TestRemoveMember_LastAdmin_Returns403(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-1"
	adminID := "admin-1"

	env.memberRepo.Members[orgID+":"+adminID] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   adminID,
		Role:     "org_admin",
		JoinedAt: time.Now(),
	}

	token := generateTestJWT(adminID, orgID, "org_admin")

	req, _ := http.NewRequest(http.MethodDelete, "/v1/organizations/"+orgID+"/members/"+adminID, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "cannot remove the last organization admin")
}

func TestRemoveMember_Success_WhenMultipleAdmins(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-1"
	admin1 := "admin-1"
	admin2 := "admin-2"

	env.memberRepo.Members[orgID+":"+admin1] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   admin1,
		Role:     "org_admin",
		JoinedAt: time.Now(),
	}
	env.memberRepo.Members[orgID+":"+admin2] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   admin2,
		Role:     "org_admin",
		JoinedAt: time.Now(),
	}

	token := generateTestJWT(admin1, orgID, "org_admin")

	req, _ := http.NewRequest(http.MethodDelete, "/v1/organizations/"+orgID+"/members/"+admin2, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, env.auditWriter.HasAction("member_removed"))
}

func TestUpdateRole_SelfDemotion_LastAdmin_Returns403(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-1"
	adminID := "admin-1"

	env.memberRepo.Members[orgID+":"+adminID] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   adminID,
		Role:     "org_admin",
		JoinedAt: time.Now(),
	}

	token := generateTestJWT(adminID, orgID, "org_admin")

	body := dto.UpdateRoleRequest{
		Role: "staff",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/v1/organizations/"+orgID+"/members/"+adminID+"/role", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "cannot demote the last organization admin")
}

func TestUpdateRole_Success(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-1"
	staffID := "staff-1"
	adminID := "admin-1"

	env.memberRepo.Members[orgID+":"+staffID] = &model.OrgMember{
		OrgID:    orgID,
		UserID:   staffID,
		Role:     "staff",
		JoinedAt: time.Now(),
	}

	token := generateTestJWT(adminID, orgID, "org_admin")

	body := dto.UpdateRoleRequest{
		Role: "finance",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPut, "/v1/organizations/"+orgID+"/members/"+staffID+"/role", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "finance", env.memberRepo.Members[orgID+":"+staffID].Role)
	assert.True(t, env.auditWriter.HasAction("role_updated"))
}

func TestCreateInvitation_Success(t *testing.T) {
	env := setupTestEnv()
	orgID := "org-1"
	adminID := "admin-1"
	token := generateTestJWT(adminID, orgID, "org_admin")

	body := dto.InviteRequest{
		Email: "invitee@example.com",
		Role:  "staff",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/v1/organizations/"+orgID+"/invitations", bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]dto.InviteResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	inv := resp["data"]
	assert.NotEmpty(t, inv.Token)
	assert.Equal(t, 64, len(inv.Token)) // 32 bytes hex encoded = 64 chars
	assert.Equal(t, "invitee@example.com", inv.Email)
	assert.Equal(t, "staff", inv.Role)
	assert.True(t, inv.ExpiresAt.After(time.Now()))
	assert.True(t, env.auditWriter.HasAction("invite_created"))
}

func TestAcceptInvitation_ExpiredToken(t *testing.T) {
	env := setupTestEnv()
	testToken := "expired-token-12345"

	env.invitationRepo.Invitations[testToken] = &model.Invitation{
		ID:        "inv-1",
		OrgID:     "org-1",
		Email:     "user@example.com",
		Role:      "staff",
		Token:     testToken,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}

	body := dto.AcceptInviteRequest{
		Name:     "Test User",
		Password: "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/v1/invitations/"+testToken+"/accept", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "expired")
}

func TestAcceptInvitation_NewUser_Success(t *testing.T) {
	env := setupTestEnv()
	testToken := "valid-token-12345"

	env.invitationRepo.Invitations[testToken] = &model.Invitation{
		ID:        "inv-1",
		OrgID:     "org-1",
		Email:     "newuser@example.com",
		Role:      "finance",
		Token:     testToken,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
	}

	body := dto.AcceptInviteRequest{
		Name:     "New Finance User",
		Password: "strongpassword123",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/v1/invitations/"+testToken+"/accept", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]dto.MemberResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	member := resp["data"]
	assert.NotEmpty(t, member.UserID)
	assert.Equal(t, "newuser@example.com", member.Email)
	assert.Equal(t, "finance", member.Role)

	// Verify user registered
	user, exists := env.userRepo.Users["newuser@example.com"]
	assert.True(t, exists)
	assert.NotEmpty(t, user.PasswordHash)

	// Verify member inserted
	m, exists := env.memberRepo.Members["org-1:"+member.UserID]
	assert.True(t, exists)
	assert.Equal(t, "finance", m.Role)

	// Verify invitation marked used
	inv := env.invitationRepo.Invitations[testToken]
	assert.NotNil(t, inv.UsedAt)

	// Verify audit log
	assert.True(t, env.auditWriter.HasAction("invite_accepted"))
}
