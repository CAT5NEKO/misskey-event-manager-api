package handler

import (
	"encoding/json"
	"net/http"

	"miSchedule/internal/middleware"
	"miSchedule/internal/model"
	"miSchedule/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input model.AuthLoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "リクエスト形式が不正です")
		return
	}
	if input.Host == "" {
		respondError(w, http.StatusBadRequest, "Misskeyサーバーを入力してください")
		return
	}

	resp, err := h.authService.Login(input.Host)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	var input model.AuthCallbackInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "リクエスト形式が不正です")
		return
	}

	ipAddress := middleware.GetIPAddress(r)
	userAgent := r.UserAgent()

	resp, err := h.authService.Callback(input.Session, input.CSRFToken, ipAddress, userAgent)
	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var input model.RefreshInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "リクエスト形式が不正です")
		return
	}

	ipAddress := middleware.GetIPAddress(r)
	userAgent := r.UserAgent()

	resp, err := h.authService.Refresh(input.RefreshToken, ipAddress, userAgent)
	if err != nil {
		status := http.StatusUnauthorized
		if err.Error() == "refresh token reuse detected" {
			status = http.StatusForbidden
		}
		respondError(w, status, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "認証が必要です")
		return
	}

	user, err := h.authService.GetUser(userID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "認証が必要です")
		return
	}

	ipAddress := middleware.GetIPAddress(r)
	userAgent := r.UserAgent()

	if err := h.authService.Revoke(userID, ipAddress, userAgent); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "token revoked"})
}

func (h *AuthHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "認証が必要です")
		return
	}

	ipAddress := middleware.GetIPAddress(r)
	userAgent := r.UserAgent()

	if err := h.authService.DeleteAccount(userID, ipAddress, userAgent); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "account deleted"})
}
