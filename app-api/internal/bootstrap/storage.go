package bootstrap

import (
	"context"
	"fmt"
	"time"

	"prasankit-api/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const storageConnectTimeout = 5 * time.Second

func OpenMinIO(cfg config.Config) (*minio.Client, error) {
	client, err := minio.New(cfg.Storage.EndpointHost, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.Storage.AccessKey, cfg.Storage.SecretKey, ""),
		Secure: cfg.Storage.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), storageConnectTimeout)
	defer cancel()

	if _, err := client.ListBuckets(ctx); err != nil {
		return nil, fmt.Errorf("check minio connection: %w", err)
	}

	return client, nil
}
