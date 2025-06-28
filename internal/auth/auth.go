package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type CustomClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Plan   string `json:"plan"`
	jwt.RegisteredClaims
}

func CreateJWT(userID, role, plan, jwtSecret string, expiry time.Duration) (string, error) {
	claims := CustomClaims{
		UserID: userID,
		Role:   role,
		Plan:   plan,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func VerifyJWT(tokenStr, jwtSecret string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}
