package auth

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

type Authenticator struct {
	aud    string
	iss    string
	secret string
}

func NewAuthenticator(secret, aud, iss string) *Authenticator {
	return &Authenticator{
		secret: secret,
		aud:    aud,
		iss:    iss,
	}
}

func (au *Authenticator) GenerateToken(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(au.secret))

	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (au *Authenticator) ValidateToken(token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(au.secret), nil
	},
		jwt.WithAudience(au.aud),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(au.iss),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
	)
}
