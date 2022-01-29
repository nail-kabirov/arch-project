package app

import (
	"arch-homework/pkg/common/app/uuid"

	"errors"
)

var ErrUserNotFound = errors.New("user not found")
var ErrInvalidEmail = errors.New("email is invalid")
var ErrEmailAlreadyExists = errors.New("email already exists")

type UserID uuid.UUID
type Email string
type Address string

type UserProfile struct {
	UserID    UserID
	Login     string
	FirstName string
	LastName  string
	Email     Email
	Address   Address
}

type UserProfileRepositoryRead interface {
	FindByID(id UserID) (*UserProfile, error)
	FindByEmail(email Email) (*UserProfile, error)
}

type UserProfileRepository interface {
	UserProfileRepositoryRead
	Store(user *UserProfile) error
	Remove(id UserID) error
}
