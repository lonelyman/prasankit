// Package session implements auth.SessionStore against Redis.
// Key format: "session:" + sha256(raw_token).
package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"prasankit-api/internal/modules/auth"

	"github.com/redis/go-redis/v9"
)

const keyPrefix = "session:"

// Store implements auth.SessionStore using Redis.
type Store struct {
	client *redis.Client
}

// NewStore constructs a Store backed by the given Redis client.
func NewStore(client *redis.Client) *Store {
	return &Store{client: client}
}

// Create stores the session record under "session:<hash>" with the given TTL.
func (s *Store) Create(ctx context.Context, hash string, record auth.SessionRecord, ttl time.Duration) error {
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("sessionstore: marshal record: %w", err)
	}
	if err := s.client.Set(ctx, key(hash), data, ttl).Err(); err != nil {
		return fmt.Errorf("sessionstore: set: %w", err)
	}
	return nil
}

// Get retrieves a session record by hash. Returns nil, nil when not found (expired or deleted).
func (s *Store) Get(ctx context.Context, hash string) (*auth.SessionRecord, error) {
	data, err := s.client.Get(ctx, key(hash)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("sessionstore: get: %w", err)
	}
	var rec auth.SessionRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, fmt.Errorf("sessionstore: unmarshal record: %w", err)
	}
	return &rec, nil
}

// Delete removes the session. Idempotent — missing key is not an error.
func (s *Store) Delete(ctx context.Context, hash string) error {
	if err := s.client.Del(ctx, key(hash)).Err(); err != nil {
		return fmt.Errorf("sessionstore: delete: %w", err)
	}
	return nil
}

func key(hash string) string {
	return keyPrefix + hash
}
