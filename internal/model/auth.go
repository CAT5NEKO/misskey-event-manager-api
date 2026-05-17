package model

import "time"

type MiAuthUser struct {
	ID        string  `json:"id"`
	Username  string  `json:"username"`
	Name      *string `json:"name"`
	AvatarURL *string `json:"avatarUrl"`
	Host      *string `json:"host"`
}

type MiAuthCheckResponse struct {
	OK    bool       `json:"ok"`
	Token string     `json:"token"`
	User  MiAuthUser `json:"user"`
}

type MiAuthSession struct {
	ID        string    `json:"id"`
	Host      string    `json:"host"`
	CSRFToken  string    `json:"csrf_token"`
	MiauthURL  string    `json:"miauth_url"`
	CreatedAt time.Time `json:"-"`
}

type AuthLoginInput struct {
	Host string `json:"host"`
}

type LoginResponse struct {
	MiauthURL  string `json:"miauth_url"`
	SessionID  string `json:"session_id"`
	CSRFToken   string `json:"csrf_token"`
}

type AuthCallbackInput struct {
	Session   string `json:"session"`
	CSRFToken string `json:"csrf_token"`
}

type CallbackResponse struct {
	JWT          string `json:"jwt"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type RefreshInput struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshResponse struct {
	JWT          string `json:"jwt"`
	RefreshToken string `json:"refresh_token"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}
