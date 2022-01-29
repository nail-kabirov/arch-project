package app

import (
	"arch-homework/pkg/common/app/uuid"
	"errors"
)

var ErrSessionNotFound = errors.New("user not found")

type SessionID uuid.UUID

type Session struct {
	ID     SessionID
	UserID UserID
}

type SessionClient interface {
	Store(session Session) error
	Remove(id SessionID) error
	FindByID(id SessionID) (*Session, error)
	UpdateSessionTTL(id SessionID) error
}
