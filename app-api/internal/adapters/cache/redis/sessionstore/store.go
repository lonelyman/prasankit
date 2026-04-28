package sessionstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"prasankit-api/internal/modules/auth/authsvc"

	"github.com/redis/go-redis/v9"
)

type Store struct {
	redis *redis.Client
}

func New(redisClient *redis.Client) Store {
	return Store{
		redis: redisClient,
	}
}

func (s Store) Save(ctx context.Context, session authsvc.SessionRecord, ttl time.Duration) error {
	if ttl <= 0 {
		return fmt.Errorf("session ttl must be greater than 0")
	}

	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, sessionKey(session.SessionKeyHash), payload, ttl).Err()
}

func (s Store) Get(ctx context.Context, sessionKeyHash string) (*authsvc.SessionRecord, error) {
	payload, err := s.redis.Get(ctx, sessionKey(sessionKeyHash)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, authsvc.ErrSessionNotFound
		}
		return nil, err
	}

	var session authsvc.SessionRecord
	if err := json.Unmarshal(payload, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s Store) Delete(ctx context.Context, sessionKeyHash string) error {
	return s.redis.Del(ctx, sessionKey(sessionKeyHash)).Err()
}

func sessionKey(sessionKeyHash string) string {
	return "session:auth:" + sessionKeyHash
}
