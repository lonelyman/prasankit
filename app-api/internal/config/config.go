package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	APIEnv             string
	APIPort            string
	APITimezone        string
	CORSAllowedOrigins []string
	Postgres           PostgresConfig
	Redis              RedisConfig
	Storage            StorageConfig
	Mail               MailConfig
}

// MailConfig holds SMTP transport settings for outbound email.
// SMTPHost, SMTPPort, FromAddress, InviteBaseURL, and VerifyBaseURL are required — boot fails if missing.
// Username, Password, and FromName are optional (empty = no auth / no display name).
type MailConfig struct {
	SMTPHost      string // SMTP server hostname
	SMTPPort      string // SMTP server port (string — passed to net.JoinHostPort)
	FromAddress   string // envelope sender and From: header address
	FromName      string // optional display name ("Name <addr>" when set)
	Username      string // SMTP auth username; empty = no auth (e.g. Mailpit dev)
	Password      string // SMTP auth password
	InviteBaseURL string // base URL for invitation accept links (e.g. http://localhost:13000/invitations/accept)
	VerifyBaseURL string // base URL for email verification links (e.g. http://localhost:13000/verify-email)
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// StorageConfig holds only the fields OpenMinIO uses (EndpointHost, UseSSL, AccessKey, SecretKey, Bucket).
type StorageConfig struct {
	EndpointHost string
	UseSSL       bool
	AccessKey    string
	SecretKey    string
	Bucket       string
}

func Load() (Config, error) {
	apiEnv, err := requiredEnv("API_ENV")
	if err != nil {
		return Config{}, err
	}

	apiPort, err := requiredEnv("API_PORT")
	if err != nil {
		return Config{}, err
	}

	apiTimezone, err := requiredEnv("API_TIMEZONE")
	if err != nil {
		return Config{}, err
	}

	corsRaw, err := requiredEnv("API_CORS_ALLOWED_ORIGINS")
	if err != nil {
		return Config{}, err
	}
	corsOrigins := parseCORSOrigins(corsRaw)

	postgres, err := loadPostgresConfig()
	if err != nil {
		return Config{}, err
	}

	redis, err := loadRedisConfig()
	if err != nil {
		return Config{}, err
	}

	storage, err := loadStorageConfig()
	if err != nil {
		return Config{}, err
	}

	mail, err := loadMailConfig()
	if err != nil {
		return Config{}, err
	}

	return Config{
		APIEnv:             apiEnv,
		APIPort:            apiPort,
		APITimezone:        apiTimezone,
		CORSAllowedOrigins: corsOrigins,
		Postgres:           postgres,
		Redis:              redis,
		Storage:            storage,
		Mail:               mail,
	}, nil
}

func (c Config) HTTPAddress() string {
	return net.JoinHostPort("", c.APIPort)
}

func (c Config) RedisAddress() string {
	return net.JoinHostPort(c.Redis.Host, c.Redis.Port)
}

func (c Config) PostgresDSN() string {
	dsn := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.Postgres.User, c.Postgres.Password),
		Host:   net.JoinHostPort(c.Postgres.Host, c.Postgres.Port),
		Path:   "/" + c.Postgres.Name,
	}

	query := dsn.Query()
	query.Set("sslmode", c.Postgres.SSLMode)
	dsn.RawQuery = query.Encode()

	return dsn.String()
}

func parseCORSOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func loadPostgresConfig() (PostgresConfig, error) {
	host, err := requiredEnv("POSTGRES_PRIMARY_HOST")
	if err != nil {
		return PostgresConfig{}, err
	}

	port, err := requiredEnv("POSTGRES_PRIMARY_PORT")
	if err != nil {
		return PostgresConfig{}, err
	}

	user, err := requiredEnv("POSTGRES_PRIMARY_USER")
	if err != nil {
		return PostgresConfig{}, err
	}

	password, err := requiredEnv("POSTGRES_PRIMARY_PASSWORD")
	if err != nil {
		return PostgresConfig{}, err
	}

	name, err := requiredEnv("POSTGRES_PRIMARY_NAME")
	if err != nil {
		return PostgresConfig{}, err
	}

	sslMode, err := requiredEnv("POSTGRES_SSL_MODE")
	if err != nil {
		return PostgresConfig{}, err
	}

	return PostgresConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Name:     name,
		SSLMode:  sslMode,
	}, nil
}

func loadRedisConfig() (RedisConfig, error) {
	host, err := requiredEnv("REDIS_HOST")
	if err != nil {
		return RedisConfig{}, err
	}

	port, err := requiredEnv("REDIS_PORT")
	if err != nil {
		return RedisConfig{}, err
	}

	password, err := requiredEnv("REDIS_PASSWORD")
	if err != nil {
		return RedisConfig{}, err
	}

	dbValue, err := requiredEnv("REDIS_DB")
	if err != nil {
		return RedisConfig{}, err
	}

	db, err := strconv.Atoi(dbValue)
	if err != nil {
		return RedisConfig{}, fmt.Errorf("invalid REDIS_DB: %w", err)
	}

	return RedisConfig{
		Host:     host,
		Port:     port,
		Password: password,
		DB:       db,
	}, nil
}

func loadStorageConfig() (StorageConfig, error) {
	endpoint, err := requiredEnv("STORAGE_ENDPOINT")
	if err != nil {
		return StorageConfig{}, err
	}

	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return StorageConfig{}, fmt.Errorf("invalid STORAGE_ENDPOINT: %w", err)
	}
	if endpointURL.Host == "" {
		return StorageConfig{}, fmt.Errorf("invalid STORAGE_ENDPOINT: host is required")
	}
	if endpointURL.Scheme != "http" && endpointURL.Scheme != "https" {
		return StorageConfig{}, fmt.Errorf("invalid STORAGE_ENDPOINT: scheme must be http or https")
	}

	bucket, err := requiredEnv("STORAGE_BUCKET")
	if err != nil {
		return StorageConfig{}, err
	}

	accessKey, err := requiredEnv("MINIO_ROOT_USER")
	if err != nil {
		return StorageConfig{}, err
	}

	secretKey, err := requiredEnv("MINIO_ROOT_PASSWORD")
	if err != nil {
		return StorageConfig{}, err
	}

	return StorageConfig{
		EndpointHost: endpointURL.Host,
		UseSSL:       endpointURL.Scheme == "https",
		Bucket:       bucket,
		AccessKey:    accessKey,
		SecretKey:    secretKey,
	}, nil
}

func loadMailConfig() (MailConfig, error) {
	host, err := requiredEnv("MAIL_SMTP_HOST")
	if err != nil {
		return MailConfig{}, err
	}

	port, err := requiredEnv("MAIL_SMTP_PORT")
	if err != nil {
		return MailConfig{}, err
	}

	fromAddress, err := requiredEnv("MAIL_FROM_ADDRESS")
	if err != nil {
		return MailConfig{}, err
	}

	inviteBaseURL, err := requiredEnv("MAIL_INVITE_BASE_URL")
	if err != nil {
		return MailConfig{}, err
	}

	verifyBaseURL, err := requiredEnv("MAIL_VERIFY_BASE_URL")
	if err != nil {
		return MailConfig{}, err
	}

	return MailConfig{
		SMTPHost:      host,
		SMTPPort:      port,
		FromAddress:   fromAddress,
		FromName:      os.Getenv("MAIL_FROM_NAME"),
		Username:      os.Getenv("MAIL_SMTP_USERNAME"),
		Password:      os.Getenv("MAIL_SMTP_PASSWORD"),
		InviteBaseURL: inviteBaseURL,
		VerifyBaseURL: verifyBaseURL,
	}, nil
}

func requiredEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return value, nil
}
