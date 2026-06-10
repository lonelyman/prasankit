package bootstrap

import (
	"context"
	"database/sql"

	sessionstore "prasankit-api/internal/adapters/cache/session"
	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	authdbrepo "prasankit-api/internal/adapters/database/auth"
	companypositiondbrepo "prasankit-api/internal/adapters/database/companyposition"
	deliverabledbrepo "prasankit-api/internal/adapters/database/deliverable"
	projectdbrepo "prasankit-api/internal/adapters/database/project"
	projectmemberdbrepo "prasankit-api/internal/adapters/database/projectmember"
	projectmemberpositiondbrepo "prasankit-api/internal/adapters/database/projectmemberposition"
	projectpositiondbrepo "prasankit-api/internal/adapters/database/projectposition"
	workspacedbrepo "prasankit-api/internal/adapters/database/workspace"
	smtpadapter "prasankit-api/internal/adapters/email/smtp"
	"prasankit-api/internal/config"
	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/companyposition"
	"prasankit-api/internal/modules/deliverable"
	"prasankit-api/internal/modules/project"
	"prasankit-api/internal/modules/projectmember"
	"prasankit-api/internal/modules/projectmemberposition"
	"prasankit-api/internal/modules/projectposition"
	"prasankit-api/internal/modules/workspace"
	httptransport "prasankit-api/internal/transport/http"
	authhandler "prasankit-api/internal/transport/http/auth"
	companypositionhandler "prasankit-api/internal/transport/http/companyposition"
	deliverablehandler "prasankit-api/internal/transport/http/deliverable"
	"prasankit-api/internal/transport/http/health"
	"prasankit-api/internal/transport/http/middlewares"
	projecthandler "prasankit-api/internal/transport/http/project"
	projectmemberhandler "prasankit-api/internal/transport/http/projectmember"
	projectmemberpositionhandler "prasankit-api/internal/transport/http/projectmemberposition"
	projectpositionhandler "prasankit-api/internal/transport/http/projectposition"
	workspacehandler "prasankit-api/internal/transport/http/workspace"

	"github.com/gofiber/fiber/v3"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func NewHTTPApp(
	postgres *sql.DB,
	gormDB *gorm.DB,
	redisClient *redis.Client,
	storageClient *minio.Client,
	corsAllowedOrigins []string,
	apiEnv string,
	mailCfg config.MailConfig,
	deliverableCfg config.DeliverableConfig,
) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "prasankit-api",
		ErrorHandler: middlewares.ErrorHandler,
	})

	healthHandler := health.NewHandler(map[string]health.CheckFunc{
		"postgres": postgres.PingContext,
		"redis": func(ctx context.Context) error {
			return redisClient.Ping(ctx).Err()
		},
		"minio": func(ctx context.Context) error {
			_, err := storageClient.ListBuckets(ctx)
			return err
		},
	})

	// Shared email sender (used by both auth and workspace services).
	emailSender := smtpadapter.New(smtpadapter.Config{
		Host:        mailCfg.SMTPHost,
		Port:        mailCfg.SMTPPort,
		FromAddress: mailCfg.FromAddress,
		FromName:    mailCfg.FromName,
		Username:    mailCfg.Username,
		Password:    mailCfg.Password,
	})

	// Auth wiring.
	accountRepo := authdbrepo.NewAccountRepo(gormDB)
	identityRepo := authdbrepo.NewIdentityRepo(gormDB)
	eventRepo := authdbrepo.NewSecurityEventRepo(gormDB)
	verifyRepo := authdbrepo.NewVerificationTokenRepo(gormDB)
	resetRepo := authdbrepo.NewPasswordResetTokenRepo(gormDB)
	sessStore := sessionstore.NewStore(redisClient)
	authSvc := auth.NewService(accountRepo, identityRepo, eventRepo, sessStore, verifyRepo, emailSender, mailCfg.VerifyBaseURL, resetRepo, mailCfg.ResetBaseURL)
	authH := authhandler.NewHandler(authSvc, apiEnv)

	// Workspace wiring.
	auditRepo := auditdbrepo.NewAuditRepo(gormDB)
	wsRepo := workspacedbrepo.NewWorkspaceRepo(gormDB, auditRepo)
	memberRepo := workspacedbrepo.NewMembershipRepo(gormDB)
	inviteRepo := workspacedbrepo.NewInvitationRepo(gormDB, auditRepo)
	workspaceSvc := workspace.NewService(wsRepo, memberRepo, inviteRepo, emailSender, mailCfg.InviteBaseURL)
	workspaceH := workspacehandler.NewHandler(workspaceSvc)

	// Project wiring.
	projectRepo := projectdbrepo.NewProjectRepo(gormDB, auditRepo)
	projectMasterRepo := projectdbrepo.NewMasterRepo(gormDB)
	projectSvc := project.NewService(projectRepo, projectMasterRepo)
	projectH := projecthandler.NewHandler(projectSvc)

	// Project member wiring. projectmember.NewService takes (members, masters, projects)
	// — 3 params (OPEN QUESTION option 1): the owner-read for the role-drift / owner-removal
	// guard reads OwnerProjectMemberID via the injected project.ProjectRepository.
	projectMemberRepo := projectmemberdbrepo.NewProjectMemberRepo(gormDB, auditRepo)
	projectMemberSvc := projectmember.NewService(projectMemberRepo, projectMasterRepo, projectRepo)
	projectMemberH := projectmemberhandler.NewHandler(projectMemberSvc)

	// Position masters + junction wiring (6b-2). Reuse the shared auditRepo + projectMemberRepo.
	projectPositionRepo := projectpositiondbrepo.NewProjectPositionRepo(gormDB, auditRepo)
	projectPositionMasterRepo := projectpositiondbrepo.NewMasterRepo(gormDB)
	projectPositionSvc := projectposition.NewService(projectPositionRepo)
	projectPositionH := projectpositionhandler.NewHandler(projectPositionSvc)

	companyPositionRepo := companypositiondbrepo.NewCompanyPositionRepo(gormDB, auditRepo)
	companyPositionMasterRepo := companypositiondbrepo.NewMasterRepo(gormDB)
	membershipPositionRepo := companypositiondbrepo.NewMembershipPositionRepo(gormDB, auditRepo)
	companyPositionSvc := companyposition.NewService(companyPositionRepo, companyPositionMasterRepo, membershipPositionRepo)
	companyPositionH := companypositionhandler.NewHandler(companyPositionSvc)

	// Junction service injects: junction repo, the existing projectMemberRepo (member FindByID),
	// and the projectposition master pre-check (active-only). Acyclic one-way edges.
	projectMemberPositionRepo := projectmemberpositiondbrepo.NewProjectMemberPositionRepo(gormDB, auditRepo)
	projectMemberPositionSvc := projectmemberposition.NewService(projectMemberPositionRepo, projectMemberRepo, projectPositionMasterRepo)
	projectMemberPositionH := projectmemberpositionhandler.NewHandler(projectMemberPositionSvc)

	// Deliverable wiring (M3). The repo holds the OQ-6 self-review flag (enforced under the
	// deliverable lock). ProjectAccessRepo resolves project existence + the actor's project_member
	// for the D63 service-layer authz gate. Reuse the shared auditRepo.
	deliverableRepo := deliverabledbrepo.NewDeliverableRepo(gormDB, auditRepo, deliverableCfg.ForbidSelfReview)
	deliverableMasterRepo := deliverabledbrepo.NewMasterRepo(gormDB)
	deliverableAccessRepo := deliverabledbrepo.NewProjectAccessRepo(gormDB)
	deliverableSvc := deliverable.NewService(deliverableRepo, deliverableMasterRepo, deliverableAccessRepo)
	deliverableH := deliverablehandler.NewHandler(deliverableSvc)

	httptransport.RegisterRoutes(
		app,
		healthHandler,
		corsAllowedOrigins,
		authSvc, authH,
		workspaceH, wsRepo, memberRepo,
		projectH,
		projectMemberH,
		projectPositionH,
		companyPositionH,
		projectMemberPositionH,
		companyPositionH, // companyPositionAttachH — same handler exposes set/clear
		deliverableH,
	)

	return app
}
