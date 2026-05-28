package bootstrap

import (
	"context"
	"database/sql"

	sessionstore "prasankit-api/internal/adapters/cache/session"
	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	authdbrepo "prasankit-api/internal/adapters/database/auth"
	workspacedbrepo "prasankit-api/internal/adapters/database/workspace"
	smtpadapter "prasankit-api/internal/adapters/email/smtp"
	"prasankit-api/internal/config"
	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/workspace"
	httptransport "prasankit-api/internal/transport/http"
	authhandler "prasankit-api/internal/transport/http/auth"
	"prasankit-api/internal/transport/http/health"
	"prasankit-api/internal/transport/http/middlewares"
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

	// Auth wiring.
	accountRepo := authdbrepo.NewAccountRepo(gormDB)
	identityRepo := authdbrepo.NewIdentityRepo(gormDB)
	eventRepo := authdbrepo.NewSecurityEventRepo(gormDB)
	sessStore := sessionstore.NewStore(redisClient)
	authSvc := auth.NewService(accountRepo, identityRepo, eventRepo, sessStore)
	authH := authhandler.NewHandler(authSvc, apiEnv)

	// Workspace wiring.
	auditRepo := auditdbrepo.NewAuditRepo(gormDB)
	wsRepo := workspacedbrepo.NewWorkspaceRepo(gormDB, auditRepo)
	memberRepo := workspacedbrepo.NewMembershipRepo(gormDB)
	inviteRepo := workspacedbrepo.NewInvitationRepo(gormDB, auditRepo)
	emailSender := smtpadapter.New(smtpadapter.Config{
		Host:        mailCfg.SMTPHost,
		Port:        mailCfg.SMTPPort,
		FromAddress: mailCfg.FromAddress,
		FromName:    mailCfg.FromName,
		Username:    mailCfg.Username,
		Password:    mailCfg.Password,
	})
	workspaceSvc := workspace.NewService(wsRepo, memberRepo, inviteRepo, emailSender, mailCfg.InviteBaseURL)
	workspaceH := workspacehandler.NewHandler(workspaceSvc)

	httptransport.RegisterRoutes(
		app,
		healthHandler,
		corsAllowedOrigins,
		authSvc, authH,
		workspaceH, wsRepo, memberRepo,
	)

	return app
}
