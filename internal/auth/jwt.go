package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"miSchedule/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrExpiredToken      = errors.New("token expired")
	ErrTokenReuse         = errors.New("refresh token reuse detected")
	ErrUserInactive      = errors.New("user account is deactivated")
)

type JWTManager struct {
	accessSecret  []byte
	refreshSecret []byte
}

func NewJWTManager(accessSecret, refreshSecret string) *JWTManager {
	return &JWTManager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
	}
}

type AccessClaims struct {
	UserID    string `json:"user_id"`
	IsAdmin   bool   `json:"is_admin"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserID    string `json:"user_id"`
	TokenHash string `json:"token_hash"`
	FamilyID  string `json:"family_id"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

func (m *JWTManager) GenerateAccessToken(user model.User) (string, error) {
	claims := AccessClaims{
		UserID:    user.ID.String(),
		IsAdmin:   user.IsAdmin,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "miSchedule",
			Subject:   user.ID.String(),
			Audience:  jwt.ClaimStrings{"miSchedule"},
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.accessSecret)
}

func (m *JWTManager) GenerateRefreshToken(userID uuid.UUID, tokenHash, familyID string) (string, error) {
	claims := RefreshClaims{
		UserID:    userID.String(),
		TokenHash: tokenHash,
		FamilyID:  familyID,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "miSchedule",
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{"miSchedule"},
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.refreshSecret)
}

func (m *JWTManager) ValidateAccessToken(tokenStr string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &AccessClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.accessSecret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*AccessClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (m *JWTManager) ValidateRefreshToken(tokenStr string) (*RefreshClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &RefreshClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.refreshSecret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*RefreshClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	if claims.TokenType != "refresh" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
