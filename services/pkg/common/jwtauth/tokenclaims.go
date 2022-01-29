package jwtauth

import "github.com/golang-jwt/jwt/v4"

type TokenData interface {
	UserID() string
	UserLogin() string
}

type tokenClaims struct {
	jwt.RegisteredClaims

	ID    string `json:"uid"`
	Login string `json:"login"`
}

func (t *tokenClaims) UserID() string {
	return t.ID
}

func (t *tokenClaims) UserLogin() string {
	return t.Login
}
