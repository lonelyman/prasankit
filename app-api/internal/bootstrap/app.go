package bootstrap

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"prasankit-api/internal/adapters/cache/redis/ratelimit"
	"prasankit-api/internal/adapters/cache/redis/sessionstore"
	"prasankit-api/internal/adapters/database/postgres/authrepo"
	"prasankit-api/internal/adapters/database/postgres/workspacerepo"
	"prasankit-api/internal/adapters/email/smtpemail"
	"prasankit-api/internal/config"
	"prasankit-api/internal/modules/auth/authsvc"
	"prasankit-api/internal/modules/workspace/workspacesvc"
	"prasankit-api/internal/transport/http/authhttp"
	"prasankit-api/internal/transport/http/workspacehttp"
	"prasankit-api/pkg/passwordhash"

	"github.com/gofiber/fiber/v3"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const shutdownTimeout = 10 * time.Second

type App struct {
	config      config.Config
	httpApp     *fiber.App
	postgres    *gorm.DB
	postgresSQL *sql.DB
	redis       *redis.Client
	storage     *minio.Client
}

func InitializeApp(cfg config.Config) (*App, error) {
	postgresDB, postgresSQLDB, err := OpenPostgres(cfg)
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

	authRepository := authrepo.NewRepository(postgresDB)
	authService := authsvc.NewService(
		authRepository,
		passwordhash.NewBcryptHasher(0),
		smtpemail.NewSender(cfg.Mail),
		ratelimit.New(redisClient),
		sessionstore.New(redisClient),
		authsvc.ServiceConfig{
			VerificationBaseURL:   cfg.Mail.VerifyBaseURL,
			VerificationTokenTTL:  cfg.Mail.VerifyTokenTTL,
			VerificationIPLimit:   cfg.Mail.VerifyIPLimit,
			VerificationIPWindow:  cfg.Mail.VerifyIPWindow,
			SessionSecret:         cfg.Session.Secret,
			SessionTTL:            cfg.Session.TTL,
			PasswordResetBaseURL:  cfg.Mail.ResetBaseURL,
			PasswordResetTokenTTL: cfg.Mail.ResetTokenTTL,
			PasswordResetIPLimit:  cfg.Mail.ResetIPLimit,
			PasswordResetIPWindow: cfg.Mail.ResetIPWindow,
		},
	)
	authHandler := authhttp.NewHandler(authService, authhttp.CookieConfig{
		Name:     cfg.Session.CookieName,
		TTL:      cfg.Session.TTL,
		Secure:   cfg.Cookie.Secure,
		SameSite: cfg.Cookie.SameSite,
	})
	workspaceRepository := workspacerepo.NewRepository(postgresDB)
	workspaceService := workspacesvc.NewService(workspaceRepository)
	workspaceHandler := workspacehttp.NewHandler(workspaceService, authService, workspacehttp.CookieConfig{
		Name:     cfg.Session.CookieName,
		TTL:      cfg.Session.TTL,
		Secure:   cfg.Cookie.Secure,
		SameSite: cfg.Cookie.SameSite,
	})

	return &App{
		config:      cfg,
		httpApp:     NewHTTPApp(postgresSQLDB, redisClient, storageClient, authHandler, workspaceHandler),
		postgres:    postgresDB,
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
