package redis

import (
	"arch-homework/pkg/auth/app"
	"arch-homework/pkg/common/app/uuid"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"

	"context"
	"fmt"
	"time"
)

func NewSessionClient(client *redis.Client, sessionTTL time.Duration) app.SessionClient {
	return &sessionClient{
		client:     client,
		sessionTTL: sessionTTL,
	}
}

type sessionClient struct {
	client     *redis.Client
	sessionTTL time.Duration
}

func (s *sessionClient) Store(session app.Session) error {
	value := string(session.UserID)
	_, err := s.client.Set(context.Background(), sessionKey(session.ID), value, s.sessionTTL).Result()
	return errors.WithStack(err)
}

func (s *sessionClient) Remove(id app.SessionID) error {
	_, err := s.client.Del(context.Background(), sessionKey(id)).Result()
	return errors.WithStack(err)
}

func (s *sessionClient) FindByID(id app.SessionID) (*app.Session, error) {
	data, err := s.client.Get(context.Background(), sessionKey(id)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) || data == "" {
			return nil, app.ErrSessionNotFound
		}
		return nil, errors.WithStack(err)
	}
	err = uuid.ValidateUUID(data)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	session := app.Session{
		ID:     id,
		UserID: app.UserID(data),
	}
	return &session, nil
}

func (s *sessionClient) UpdateSessionTTL(id app.SessionID) error {
	_, err := s.client.Expire(context.Background(), sessionKey(id), s.sessionTTL).Result()
	return errors.WithStack(err)
}

func sessionKey(id app.SessionID) string {
	return fmt.Sprintf("session:%s", string(id))
}
