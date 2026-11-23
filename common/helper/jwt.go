package helper

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GetJWTSecret() []byte {
	if s := os.Getenv("JWT_SECRET"); len(s) >= 32 {
		return []byte(s)
	}
	if p := os.Getenv("PASSWORD_AUTH"); p != "" {
		return []byte("unpaklink-hmac-signing-key-2026-" + p)
	}
	return []byte("unpaklink-default-strong-hmac-secret-key-9f8e7d6c5b4a3f2e1d0c")
}

type JWTService struct {
	Secret []byte
}

func GenerateToken(sid string) (string, string, error) {
	secret := GetJWTSecret()

	// Access token (24 jam)
	accessClaims := jwt.MapClaims{
		"sid": sid,
		"exp": time.Now().Add(1 * 24 * time.Hour).Unix(),
	}
	accessJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenStr, err := accessJWT.SignedString(secret)
	if err != nil {
		return "", "", err
	}

	// Refresh token (14 hari)
	refreshClaims := jwt.MapClaims{
		"sid": sid,
		"exp": time.Now().Add(14 * 24 * time.Hour).Unix(),
	}
	refreshJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshJWT.SignedString(secret)
	if err != nil {
		return "", "", err
	}

	return accessTokenStr, refreshTokenStr, nil
}
