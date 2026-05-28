package bootstrap

import (
	"context"
	"database/sql"

	sessionstore "prasankit-api/internal/adapters/cache/session"
	authdbrepo "prasankit-api/internal/adapters/database/auth"
	"prasankit-api/internal/modules/auth"
	httptransport "prasankit-api/internal/transport/http"
	authhandler "prasankit-api/internal/transport/http/auth"
	"prasankit-api/internal/transport/http/health"
	"prasankit-api/internal/transport/http/middlewares"

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

	httptransport.RegisterRoutes(app, healthHandler, corsAllowedOrigins, authSvc, authH)

	return app
}
