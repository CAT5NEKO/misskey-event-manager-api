package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID              uuid.UUID  `json:"id"`
	MisskeyUserID   string     `json:"-"`
	MisskeyUsername string     `json:"misskey_username"`
	MisskeyHost     string     `json:"misskey_host"`
	MisskeyToken    string     `json:"-"`
	Name            string     `json:"name"`
	AvatarURL       *string    `json:"avatar_url"`
	IsAdmin         bool       `json:"is_admin"`
	IsActive        bool       `json:"is_active"`
	LastLoginAt     *time.Time `json:"last_login_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type RefreshToken struct {
	ID        uuid.UUID `json:"-"`
	UserID    uuid.UUID `json:"-"`
	TokenHash string    `json:"-"`
	FamilyID  string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"-"`
	CreatedAt time.Time `json:"-"`
}

type PaginatedUsers struct {
	Users      []User `json:"users"`
	TotalCount int    `json:"total_count"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
}
