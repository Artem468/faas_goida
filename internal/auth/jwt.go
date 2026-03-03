package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type accessClaims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func generateAccessToken(userID int64, secret string, expiresAt time.Time) (string, error) {
	claims := accessClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
