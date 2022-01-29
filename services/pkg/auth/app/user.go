package app

import (
	"arch-homework/pkg/common/app/uuid"
	"errors"
)

var ErrUserNotFound = errors.New("user not found")
var ErrLoginTooLong = errors.New("login too long")
var ErrLoginAlreadyExists = errors.New("login already exists")
var ErrInvalidPassword = errors.New("invalid password")

type UserID uuid.UUID
type Login string
type Password string

type User struct {
	UserID   UserID
	Login    Login
	Password Password
}

type UserRepositoryRead interface {
	FindByID(id UserID) (*User, error)
	FindByLogin(login Login) (*User, error)
}

type UserRepository interface {
	UserRepositoryRead
	Store(user *User) error
	Remove(id UserID) error
}
