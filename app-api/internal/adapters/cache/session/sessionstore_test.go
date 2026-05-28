package session_test

import (
	"context"
	"os"
	"testing"
	"time"

	"prasankit-api/internal/adapters/cache/session"
	"prasankit-api/internal/modules/auth"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// redisClient builds a Redis client from the same env vars config.Load uses.
func redisClient(t *testing.T) *redis.Client {
	t.Helper()
	host := envOr("REDIS_HOST", "localhost")
	port := envOr("REDIS_PORT", "16380")
	password := envOr("REDIS_PASSWORD", "change_me")

	client := redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: password,
		DB:       0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not reachable (%v): skipping integration test", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func TestSessionStore_CreateGetDelete(t *testing.T) {
	rc := redisClient(t)
	store := session.NewStore(rc)
	ctx := context.Background()

	accountID := uuid.MustParse("01900000-0000-7000-8000-000000000001")
	now := time.Now().UTC().Truncate(time.Second)

	rec := auth.SessionRecord{
		AccountID:  accountID,
		CreatedAt:  now,
		LastSeenAt: now,
		IP:         "127.0.0.1",
		UserAgent:  "test-agent",
	}

	hash := "test-session-hash-" + uuid.NewString()

	// Create.
	if err := store.Create(ctx, hash, rec, 10*time.Second); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get — should return the record.
	got, err := store.Get(ctx, hash)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil, want record")
	}
	if got.AccountID != accountID {
		t.Errorf("AccountID = %v, want %v", got.AccountID, accountID)
	}
	if !got.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, now)
	}

	// Delete.
	if err := store.Delete(ctx, hash); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Get after delete — should return nil.
	got, err = store.Get(ctx, hash)
	if err != nil {
		t.Fatalf("Get after delete: %v", err)
	}
	if got != nil {
		t.Error("Get after delete should return nil")
	}
}

func TestSessionStore_TTLExpiry(t *testing.T) {
	rc := redisClient(t)
	store := session.NewStore(rc)
	ctx := context.Background()

	hash := "test-expiry-" + uuid.NewString()
	rec := auth.SessionRecord{
		AccountID: uuid.MustParse("01900000-0000-7000-8000-000000000002"),
	}

	// Create with 1-second TTL.
	if err := store.Create(ctx, hash, rec, 1*time.Second); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Should be present immediately.
	got, err := store.Get(ctx, hash)
	if err != nil || got == nil {
		t.Fatalf("Get before expiry: err=%v, got=%v", err, got)
	}

	// Wait for TTL to expire.
	time.Sleep(1500 * time.Millisecond)

	got, err = store.Get(ctx, hash)
	if err != nil {
		t.Fatalf("Get after expiry: %v", err)
	}
	if got != nil {
		t.Error("session should be expired/nil after TTL")
	}
}

func TestSessionStore_DeleteIdempotent(t *testing.T) {
	rc := redisClient(t)
	store := session.NewStore(rc)
	ctx := context.Background()

	hash := "test-idem-" + uuid.NewString()
	// Delete a key that was never created — should not error.
	if err := store.Delete(ctx, hash); err != nil {
		t.Errorf("Delete of non-existent key: %v", err)
	}
}
