package jwtauth

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
)

const tokenExpirationTime = time.Minute

type TokenGenerator interface {
	GenerateToken(userID, userLogin string) (string, error)
}

func NewTokenGenerator(key string) TokenGenerator {
	return &tokenGenerator{key: []byte(key)}
}

type tokenGenerator struct {
	key []byte
}

func (t *tokenGenerator) GenerateToken(userID, userLogin string) (string, error) {
	issuedAt := jwt.NumericDate{Time: time.Now()}
	expiresAt := jwt.NumericDate{Time: time.Now().Add(tokenExpirationTime)}
	claims := tokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  &issuedAt,
			ExpiresAt: &expiresAt,
		},
		ID:    userID,
		Login: userLogin,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(t.key)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return tokenStr, nil
}
