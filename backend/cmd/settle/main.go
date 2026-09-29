package main

import (
	"log"

	authHandler "github.com/Firakef1/settle/backend/internal/auth/handler"
	authRepo "github.com/Firakef1/settle/backend/internal/auth/repository"
	authService "github.com/Firakef1/settle/backend/internal/auth/service"
	orgHandler "github.com/Firakef1/settle/backend/internal/organaization/handler"
	orgRepo "github.com/Firakef1/settle/backend/internal/organaization/repository"
	orgService "github.com/Firakef1/settle/backend/internal/organaization/service"
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
func SetupAuthHandler() *authHandler.AuthHandler {
	hashSvc := sharedService.NewHashService()
	jwtSvc := sharedService.NewJWTService()

	// Initialize repositories
	userRepo := authRepo.NewUserRepo(database.DB)
	refreshTokenRepo := authRepo.NewRefreshTokenRepo(database.DB)

	// Initialize services
	authSvc := authService.NewAuthService(userRepo, refreshTokenRepo, hashSvc, jwtSvc)

	// Initialize handlers
	return authHandler.NewAuthHandler(authSvc)
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

	// 4. Initialize Domains
	authH := SetupAuthHandler()
	orgH, memberH, invitationH, auditH, dashboardH, billingH := SetupOrgHandlers()

	allHandlers := &router.Handlers{
		Auth:       authH,
		Org:        orgH,
		Member:     memberH,
		Invitation: invitationH,
		Audit:      auditH,
		Dashboard:  dashboardH,
		Billing:    billingH,
	}

	// 4. Setup Router
	engine := router.SetupRouter(allHandlers)

	port := config.AppConfig.HTTPPort
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	// 5. Start Server
	log.Printf("Starting Settle API server on port %s...", port)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
