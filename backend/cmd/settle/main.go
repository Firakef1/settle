package main

import (
	"context"
	"log"
	"time"

	"github.com/Firakef1/settle/backend/internal/approvals"
	authHandler "github.com/Firakef1/settle/backend/internal/auth/handler"
	authRepo "github.com/Firakef1/settle/backend/internal/auth/repository"
	authService "github.com/Firakef1/settle/backend/internal/auth/service"
	orgHandler "github.com/Firakef1/settle/backend/internal/organaization/handler"
	orgRepo "github.com/Firakef1/settle/backend/internal/organaization/repository"
	orgService "github.com/Firakef1/settle/backend/internal/organaization/service"
	reqDtoPkg "github.com/Firakef1/settle/backend/internal/requests/dto"
	reqHandler "github.com/Firakef1/settle/backend/internal/requests/handler"
	reqRepoPkg "github.com/Firakef1/settle/backend/internal/requests/repository"
	reqServicePkg "github.com/Firakef1/settle/backend/internal/requests/service"
	"github.com/Firakef1/settle/backend/internal/router"
	"github.com/Firakef1/settle/backend/internal/shared/config"
	"github.com/Firakef1/settle/backend/internal/shared/database"
	sharedService "github.com/Firakef1/settle/backend/internal/shared/service"
	"github.com/joho/godotenv"

	_ "github.com/Firakef1/settle/backend/docs"
)

// @title           Settle API
// @version         1.0
// @description     Settle backend REST API.
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     Enter your Bearer token in the format: Bearer {token}

// SetupAuthHandler initializes the auth domain dependencies
func SetupAuthHandler(ctx context.Context) (*authHandler.AuthHandler, *authHandler.VerificationHandler) {
	cfg := config.AppConfig.Email

	hashSvc := sharedService.NewHashService()
	jwtSvc := sharedService.NewJWTService()
	emailSvc := sharedService.NewEmailService()

	// Initialize repositories
	userRepo := authRepo.NewUserRepo(database.DB)
	refreshTokenRepo := authRepo.NewRefreshTokenRepo(database.DB)
	verCodeRepo := authRepo.NewVerificationCodeRepo(database.DB)
	pwdResetOTPRepo := authRepo.NewPasswordResetOTPRepo(database.DB)

	// Initialize services
	authSvc := authService.NewAuthService(userRepo, refreshTokenRepo, pwdResetOTPRepo, emailSvc, hashSvc, jwtSvc)
	verSvc := authService.NewVerificationService(verCodeRepo, userRepo, emailSvc, authSvc, cfg.CodeTTL)

	// Cleanup expired verification codes in background
	go verSvc.RunCleanup(ctx, time.Hour)

	// Initialize handlers
	authH := authHandler.NewAuthHandler(authSvc, verSvc)
	verH := authHandler.NewVerificationHandler(verSvc, authSvc)

	return authH, verH
}

// SetupOrgHandlers initializes the organization domain dependencies
func SetupOrgHandlers() (
	*orgHandler.OrgHandler,
	*orgHandler.MemberHandler,
	*orgHandler.InvitationHandler,
	*orgHandler.AuditHandler,
	*orgHandler.DashboardHandler,
	*orgHandler.BillingHandler,
) {
	// 1. Initialize repositories and SQLTxRunner using database.DB
	txRunner := orgService.NewSQLTxRunner(database.DB)
	_ = txRunner
	oRepo := orgRepo.NewOrgRepository()
	mRepo := orgRepo.NewMemberRepository()
	iRepo := orgRepo.NewInvitationRepository()
	aRepo := orgRepo.NewAuditRepository()
	sRepo := orgRepo.NewStatsRepository()
	uRepo := orgRepo.NewUserRepository()

	// 2. Initialize services
	orgSvc := orgService.NewOrgService(database.DB, oRepo, mRepo, aRepo)
	memberSvc := orgService.NewMemberService(database.DB, mRepo, aRepo)
	invitationSvc := orgService.NewInvitationService(database.DB, iRepo, mRepo, uRepo, aRepo)
	auditSvc := orgService.NewAuditService(database.DB, aRepo, mRepo)
	dashboardSvc := orgService.NewDashboardService(database.DB, oRepo, mRepo, sRepo)
	billingSvc := orgService.NewBillingService(orgSvc)

	// 3. Initialize handlers
	orgH := orgHandler.NewOrgHandler(orgSvc)
	memberH := orgHandler.NewMemberHandler(memberSvc)
	invitationH := orgHandler.NewInvitationHandler(invitationSvc)
	auditH := orgHandler.NewAuditHandler(auditSvc)
	dashboardH := orgHandler.NewDashboardHandler(dashboardSvc)
	billingH := orgHandler.NewBillingHandler(billingSvc)

	return orgH, memberH, invitationH, auditH, dashboardH, billingH
}

type userProviderAdapter struct {
	repo authRepo.UserRepository
}

func (a *userProviderAdapter) GetUserBasicInfo(ctx context.Context, userID string) (reqDtoPkg.RequesterResponse, error) {
	u, err := a.repo.FindByID(ctx, userID)
	if err != nil {
		return reqDtoPkg.RequesterResponse{ID: userID}, err
	}
	return reqDtoPkg.RequesterResponse{
		ID:    u.ID,
		Name:  u.Name,
		Email: u.Email,
	}, nil
}

// SetupRequestHandlers initializes the requests domain dependencies
func SetupRequestHandlers() *reqHandler.RequestHandler {
	reqRepo := reqRepoPkg.NewRequestRepo(database.DB)
	recRepo := reqRepoPkg.NewReceiptRepo(database.DB)
	comRepo := reqRepoPkg.NewCommentRepo(database.DB)

	adapter := &userProviderAdapter{repo: authRepo.NewUserRepo(database.DB)}
	reqSvc := reqServicePkg.NewRequestService(reqRepo, recRepo, comRepo, adapter)
	return reqHandler.NewRequestHandler(reqSvc)
}

func main() {
	// 1. Load Environment
	_ = godotenv.Load()
	config.Load()

	// 2. Run Database Migrations
	if err := database.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// 3. Connect Database
	if err := database.Connect(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Root context for background goroutines (cleanup, etc.)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 4. Initialize Domains
	authH, verH := SetupAuthHandler(ctx)
	orgH, memberH, invitationH, auditH, dashboardH, billingH := SetupOrgHandlers()
	approvalH := approvals.SetupHandler(database.DB)
	requestH := SetupRequestHandlers()

	allHandlers := &router.Handlers{
		Auth:         authH,
		Verification: verH,
		Org:          orgH,
		Member:       memberH,
		Invitation:   invitationH,
		Audit:        auditH,
		Dashboard:    dashboardH,
		Billing:      billingH,
		Approval:     approvalH,
		Request:      requestH,
	}

	// 5. Setup Router
	engine := router.SetupRouter(allHandlers)

	port := config.AppConfig.HTTPPort
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	// 6. Start Server
	log.Printf("Starting Settle API server on port %s...", port)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
