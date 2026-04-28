package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	APIEnv   string
	APIPort  string
	Postgres PostgresConfig
	Redis    RedisConfig
	Storage  StorageConfig
	Mail     MailConfig
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

type StorageConfig struct {
	Endpoint       string
	EndpointHost   string
	UseSSL         bool
	PublicEndpoint string
	Bucket         string
	AccessKey      string
	SecretKey      string
	PresignTTL     time.Duration
}

type MailConfig struct {
	SMTPHost           string
	SMTPPort           string
	SMTPUsername       string
	SMTPPassword       string
	FromAddress        string
	FromName           string
	VerifyBaseURL      string
	VerifyTokenTTL     time.Duration
	VerifyIPLimit      int
	VerifyIPWindow     time.Duration
	VerifyEmailSubject string
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
		APIEnv:   apiEnv,
		APIPort:  apiPort,
		Postgres: postgres,
		Redis:    redis,
		Storage:  storage,
		Mail:     mail,
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

	publicEndpoint, err := requiredEnv("STORAGE_PUBLIC_ENDPOINT")
	if err != nil {
		return StorageConfig{}, err
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

	presignTTLValue, err := requiredEnv("STORAGE_PRESIGN_TTL")
	if err != nil {
		return StorageConfig{}, err
	}

	presignTTL, err := time.ParseDuration(presignTTLValue)
	if err != nil {
		return StorageConfig{}, fmt.Errorf("invalid STORAGE_PRESIGN_TTL: %w", err)
	}

	return StorageConfig{
		Endpoint:       endpoint,
		EndpointHost:   endpointURL.Host,
		UseSSL:         endpointURL.Scheme == "https",
		PublicEndpoint: publicEndpoint,
		Bucket:         bucket,
		AccessKey:      accessKey,
		SecretKey:      secretKey,
		PresignTTL:     presignTTL,
	}, nil
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

func loadMailConfig() (MailConfig, error) {
	smtpHost, err := requiredEnv("MAIL_SMTP_HOST")
	if err != nil {
		return MailConfig{}, err
	}

	smtpPort, err := requiredEnv("MAIL_SMTP_PORT")
	if err != nil {
		return MailConfig{}, err
	}

	smtpUsername, err := requiredEnv("MAIL_SMTP_USERNAME")
	if err != nil {
		return MailConfig{}, err
	}

	smtpPassword, err := requiredEnv("MAIL_SMTP_PASSWORD")
	if err != nil {
		return MailConfig{}, err
	}

	fromAddress, err := requiredEnv("MAIL_FROM_ADDRESS")
	if err != nil {
		return MailConfig{}, err
	}

	fromName, err := requiredEnv("MAIL_FROM_NAME")
	if err != nil {
		return MailConfig{}, err
	}

	verifyBaseURL, err := requiredEnv("MAIL_VERIFY_BASE_URL")
	if err != nil {
		return MailConfig{}, err
	}
	if _, err := url.ParseRequestURI(verifyBaseURL); err != nil {
		return MailConfig{}, fmt.Errorf("invalid MAIL_VERIFY_BASE_URL: %w", err)
	}

	verifyTokenTTLValue, err := requiredEnv("MAIL_VERIFY_TOKEN_TTL")
	if err != nil {
		return MailConfig{}, err
	}
	verifyTokenTTL, err := time.ParseDuration(verifyTokenTTLValue)
	if err != nil {
		return MailConfig{}, fmt.Errorf("invalid MAIL_VERIFY_TOKEN_TTL: %w", err)
	}

	verifyIPLimitValue, err := requiredEnv("MAIL_VERIFY_IP_LIMIT")
	if err != nil {
		return MailConfig{}, err
	}
	verifyIPLimit, err := strconv.Atoi(verifyIPLimitValue)
	if err != nil {
		return MailConfig{}, fmt.Errorf("invalid MAIL_VERIFY_IP_LIMIT: %w", err)
	}
	if verifyIPLimit < 1 {
		return MailConfig{}, fmt.Errorf("invalid MAIL_VERIFY_IP_LIMIT: must be greater than 0")
	}

	verifyIPWindowValue, err := requiredEnv("MAIL_VERIFY_IP_WINDOW")
	if err != nil {
		return MailConfig{}, err
	}
	verifyIPWindow, err := time.ParseDuration(verifyIPWindowValue)
	if err != nil {
		return MailConfig{}, fmt.Errorf("invalid MAIL_VERIFY_IP_WINDOW: %w", err)
	}

	verifyEmailSubject, err := requiredEnv("MAIL_VERIFY_EMAIL_SUBJECT")
	if err != nil {
		return MailConfig{}, err
	}

	return MailConfig{
		SMTPHost:           smtpHost,
		SMTPPort:           smtpPort,
		SMTPUsername:       smtpUsername,
		SMTPPassword:       smtpPassword,
		FromAddress:        fromAddress,
		FromName:           fromName,
		VerifyBaseURL:      verifyBaseURL,
		VerifyTokenTTL:     verifyTokenTTL,
		VerifyIPLimit:      verifyIPLimit,
		VerifyIPWindow:     verifyIPWindow,
		VerifyEmailSubject: verifyEmailSubject,
	}, nil
}

func requiredEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return value, nil
}
