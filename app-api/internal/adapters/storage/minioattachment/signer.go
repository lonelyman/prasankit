package minioattachment

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"prasankit-api/internal/modules/task"

	"github.com/minio/minio-go/v7"
)

type Signer struct {
	client *minio.Client
	bucket string
	ttl    time.Duration
	clock  func() time.Time
}

func NewSigner(client *minio.Client, bucket string, ttl time.Duration) *Signer {
	return &Signer{
		client: client,
		bucket: bucket,
		ttl:    ttl,
		clock:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *Signer) Bucket() string {
	return s.bucket
}

func (s *Signer) PresignUpload(ctx context.Context, objectKey string, _ string) (*task.AttachmentUploadURL, error) {
	ttl := s.ttl
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	url, err := s.client.PresignedPutObject(ctx, s.bucket, objectKey, ttl)
	if err != nil {
		return nil, err
	}
	return &task.AttachmentUploadURL{
		URL:       url.String(),
		ExpiresAt: s.clock().Add(ttl),
	}, nil
}

func (s *Signer) PresignDownload(ctx context.Context, objectKey string, fileName string, contentType string) (*task.AttachmentDownloadURL, error) {
	ttl := s.ttl
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	values := make(url.Values)
	if fileName != "" {
		values.Set("response-content-disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	}
	if contentType != "" {
		values.Set("response-content-type", contentType)
	}
	url, err := s.client.PresignedGetObject(ctx, s.bucket, objectKey, ttl, values)
	if err != nil {
		return nil, err
	}
	return &task.AttachmentDownloadURL{
		URL:       url.String(),
		ExpiresAt: s.clock().Add(ttl),
	}, nil
}
