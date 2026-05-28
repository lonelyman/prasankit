package bootstrap

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"prasankit-api/internal/config"

	"github.com/gofiber/fiber/v3"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
)

const shutdownTimeout = 10 * time.Second

type App struct {
	config      config.Config
	httpApp     *fiber.App
	postgresSQL *sql.DB
	redis       *redis.Client
	storage     *minio.Client
}

func InitializeApp(cfg config.Config) (*App, error) {
	gormDB, postgresSQLDB, err := OpenPostgres(cfg)
	if err != nil {
		return nil, err
	}

	redisClient, err := OpenRedis(cfg)
	if err != nil {
		_ = postgresSQLDB.Close()
		return nil, err
	}

	storageClient, err := OpenMinIO(cfg)
	if err != nil {
		_ = redisClient.Close()
		_ = postgresSQLDB.Close()
		return nil, err
	}

	return &App{
		config:      cfg,
		httpApp:     NewHTTPApp(postgresSQLDB, gormDB, redisClient, storageClient, cfg.CORSAllowedOrigins, cfg.APIEnv, cfg.Mail),
		postgresSQL: postgresSQLDB,
		redis:       redisClient,
		storage:     storageClient,
	}, nil
}

func (a *App) Run() error {
	defer a.Close()

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("starting prasankit-api on %s", a.config.HTTPAddress())
		serverErr <- a.httpApp.Listen(a.config.HTTPAddress())
	}()

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdownSignal)

	select {
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("http server stopped: %w", err)
		}
		return nil
	case sig := <-shutdownSignal:
		log.Printf("shutdown signal received: %s", sig)
	}

	if err := a.httpApp.ShutdownWithTimeout(shutdownTimeout); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	return nil
}

func (a *App) Close() {
	if a.postgresSQL != nil {
		if err := a.postgresSQL.Close(); err != nil {
			log.Printf("close postgres connection: %v", err)
		}
	}

	if a.redis != nil {
		if err := a.redis.Close(); err != nil {
			log.Printf("close redis connection: %v", err)
		}
	}
}
