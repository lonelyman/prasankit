package bootstrap

import (
	"context"
	"database/sql"

	httptransport "prasankit-api/internal/transport/http"
	"prasankit-api/internal/transport/http/authhttp"
	"prasankit-api/internal/transport/http/health"
	"prasankit-api/internal/transport/http/middlewares"
	"prasankit-api/internal/transport/http/workspacehttp"

	"github.com/gofiber/fiber/v3"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
)

func NewHTTPApp(
	postgres *sql.DB,
	redisClient *redis.Client,
	storageClient *minio.Client,
	authHandler authhttp.Handler,
	workspaceHandler workspacehttp.Handler,
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

	httptransport.RegisterRoutes(app, healthHandler, authHandler, workspaceHandler)

	return app
}
