package jwtauth

import (
	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
)

var ErrInvalidToken = errors.New("invalid token")

type TokenParser interface {
	ParseToken(token string) (TokenData, error)
}

func NewTokenParser(key string) TokenParser {
	return &tokenParser{key: []byte(key)}
}

type tokenParser struct {
	key []byte
}

func (t *tokenParser) ParseToken(token string) (TokenData, error) {
	claims := tokenClaims{}

	jwtToken, err := jwt.ParseWithClaims(
		token, &claims, func(token *jwt.Token) (i interface{}, err error) {
			return t.key, nil
		},
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if !jwtToken.Valid || jwtToken.Method.Alg() != "HS256" {
		return nil, errors.WithStack(ErrInvalidToken)
	}

	return &claims, nil
}
