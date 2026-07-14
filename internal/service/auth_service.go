package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"miSchedule/internal/auth"
	"miSchedule/internal/config"
	"miSchedule/internal/model"
	"miSchedule/internal/repository"

	"github.com/google/uuid"
)

type AuthService struct {
	cfg         *config.Config
	userRepo    *repository.UserRepo
	instanceRepo *repository.InstanceRepo
	refreshRepo *repository.RefreshTokenRepo
	settingRepo *repository.SettingRepo
	auditRepo   *repository.AuditLogRepo
	miauth      *auth.MiAuthClient
	jwtManager  *auth.JWTManager
	csrfStore   *auth.CSRFStore
	crypto      *auth.TokenCrypto
}

func NewAuthService(
	cfg *config.Config,
	userRepo *repository.UserRepo,
	instanceRepo *repository.InstanceRepo,
	refreshRepo *repository.RefreshTokenRepo,
	settingRepo *repository.SettingRepo,
	auditRepo *repository.AuditLogRepo,
	miauth *auth.MiAuthClient,
	jwtManager *auth.JWTManager,
	csrfStore *auth.CSRFStore,
	crypto *auth.TokenCrypto,
) *AuthService {
	return &AuthService{
		cfg: cfg, userRepo: userRepo, instanceRepo: instanceRepo,
		refreshRepo: refreshRepo, settingRepo: settingRepo, auditRepo: auditRepo,
		miauth: miauth, jwtManager: jwtManager, csrfStore: csrfStore, crypto: crypto,
	}
}

func (s *AuthService) Login(host string) (*model.LoginResponse, error) {
	host = strings.TrimLeft(host, "/")
	if !s.isHostAllowed(host) {
		return nil, fmt.Errorf("許可されていないインスタンスです: %s", host)
	}

	session, err := s.miauth.CreateSession(host)
	if err != nil {
		return nil, err
	}

	csrfToken := s.csrfStore.Generate(session.ID)

	return &model.LoginResponse{
		MiauthURL:  session.MiauthURL,
		SessionID:  session.ID,
		CSRFToken:  csrfToken,
	}, nil
}

func (s *AuthService) isHostAllowed(host string) bool {
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimRight(host, "/")
	host = strings.TrimLeft(host, "/")
	host = strings.TrimLeft(host, "/")

	if s.cfg.IsInstanceAllowed(host) {
		return true
	}
	allowed, _ := s.instanceRepo.IsHostAllowed(host)
	return allowed
}

func (s *AuthService) Callback(sessionID, csrfToken string, ipAddress, userAgent string) (*model.CallbackResponse, error) {
	if !s.csrfStore.Validate(sessionID, csrfToken) {
		return nil, fmt.Errorf("CSRFトークンが無効です")
	}
	s.csrfStore.Remove(sessionID)

	sessionHost := s.miauth.GetSessionHost(sessionID)
	if sessionHost == "" {
		return nil, fmt.Errorf("セッションが見つかりません")
	}

	session := &model.MiAuthSession{
		ID:        sessionID,
		Host:      sessionHost,
		CreatedAt: time.Now().Add(-5 * time.Minute),
	}

	resp, err := s.miauth.GetToken(session)
	if err != nil {
		return nil, err
	}

	userHost := session.Host
	userHost = strings.TrimPrefix(userHost, "https://")
	userHost = strings.TrimPrefix(userHost, "http://")
	userHost = strings.TrimRight(userHost, "/")
	userHost = strings.TrimLeft(userHost, "/")

	if !s.isHostAllowed(userHost) {
		return nil, fmt.Errorf("許可されていないインスタンスです: %s", userHost)
	}

	userDescription := s.getUserDisplayName(resp.User)

	encryptedToken, err := s.crypto.Encrypt(resp.Token)
	if err != nil {
		return nil, fmt.Errorf("トークンの暗号化に失敗しました: %w", err)
	}

	existingUser, err := s.userRepo.FindByMisskeyID(resp.User.ID, userHost)
	if err != nil {
		return nil, err
	}

	var localUser *model.User
	if existingUser != nil {
		localUser, err = s.updateExistingUser(existingUser, encryptedToken, userDescription, resp.User.AvatarURL)
	} else {
		localUser, err = s.createNewUser(resp.User, userHost, encryptedToken, userDescription)
	}
	if err != nil {
		return nil, err
	}

	if !localUser.IsActive {
		return nil, auth.ErrUserInactive
	}

	s.logAudit(localUser.ID, "user.login", "user", &localUser.ID, ipAddress, userAgent, nil)

	jwt, err := s.jwtManager.GenerateAccessToken(*localUser)
	if err != nil {
		return nil, err
	}

	familyID := uuid.New().String()
	refreshTokenStr, err := s.jwtManager.GenerateRefreshToken(localUser.ID, "", familyID)
	if err != nil {
		return nil, err
	}
	refreshHash := auth.HashToken(refreshTokenStr)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if err := s.refreshRepo.Create(localUser.ID, refreshHash, familyID, expiresAt); err != nil {
		return nil, err
	}

	return &model.CallbackResponse{
		JWT:          jwt,
		RefreshToken: refreshTokenStr,
		User:         *localUser,
	}, nil
}

func (s *AuthService) Refresh(refreshTokenStr, ipAddress, userAgent string) (*model.RefreshResponse, error) {
	claims, err := s.jwtManager.ValidateRefreshToken(refreshTokenStr)
	if err != nil {
		return nil, err
	}

	tokenHash := auth.HashToken(refreshTokenStr)
	userID, familyID, revoked, err := s.refreshRepo.FindByHash(tokenHash)
	if err != nil {
		return nil, err
	}
	if userID == uuid.Nil {
		return nil, auth.ErrInvalidToken
	}
	if revoked {
		s.refreshRepo.RevokeAllInFamily(familyID)
		return nil, auth.ErrTokenReuse
	}

	parsedUserID, _ := uuid.Parse(claims.UserID)
	user, err := s.userRepo.FindByID(parsedUserID)
	if err != nil || user == nil {
		return nil, auth.ErrInvalidToken
	}
	if !user.IsActive {
		return nil, auth.ErrUserInactive
	}

	s.refreshRepo.RevokeAllInFamily(familyID)

	newFamilyID := uuid.New().String()
	newRefreshStr, err := s.jwtManager.GenerateRefreshToken(user.ID, "", newFamilyID)
	if err != nil {
		return nil, err
	}
	newHash := auth.HashToken(newRefreshStr)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if err := s.refreshRepo.Create(user.ID, newHash, newFamilyID, expiresAt); err != nil {
		return nil, err
	}

	jwt, err := s.jwtManager.GenerateAccessToken(*user)
	if err != nil {
		return nil, err
	}

	s.logAudit(user.ID, "user.token_refresh", "user", &user.ID, ipAddress, userAgent, nil)

	return &model.RefreshResponse{
		JWT:          jwt,
		RefreshToken: newRefreshStr,
	}, nil
}

func (s *AuthService) Revoke(userID uuid.UUID, ipAddress, userAgent string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return fmt.Errorf("ユーザーが見つかりません")
	}

	s.userRepo.Deactivate(userID)
	s.refreshRepo.RevokeAllForUser(userID)

	s.logAudit(userID, "user.revoke_token", "user", &userID, ipAddress, userAgent, nil)
	return nil
}

func (s *AuthService) DeleteAccount(userID uuid.UUID, ipAddress, userAgent string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return fmt.Errorf("ユーザーが見つかりません")
	}

	setting, _ := s.settingRepo.Get("users.allow_self_delete")
	if setting != nil && string(setting.Value) == "false" {
		return fmt.Errorf("アカウント削除は無効化されています")
	}

	s.refreshRepo.RevokeAllForUser(userID)

	s.logAudit(userID, "user.delete", "user", &userID, ipAddress, userAgent, nil)

	return s.userRepo.Delete(userID)
}

func (s *AuthService) GetUser(userID uuid.UUID) (*model.User, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("ユーザーが見つかりません")
	}
	return user, nil
}

func (s *AuthService) logAudit(actorID uuid.UUID, action, targetType string, targetID *uuid.UUID, ipAddress, userAgent string, details map[string]interface{}) {
	var detailsBytes []byte
	if details != nil {
		detailsBytes, _ = json.Marshal(details)
	}
	s.auditRepo.Create(&model.AuditLog{
		ActorID:    actorID,
		Action:     action,
		TargetType: &targetType,
		TargetID:   targetID,
		Details:    json.RawMessage(detailsBytes),
		IPAddress:  &ipAddress,
		UserAgent:  &userAgent,
	})
}

func (s *AuthService) getUserDisplayName(user model.MiAuthUser) string {
	if user.Name != nil && *user.Name != "" {
		return *user.Name
	}
	return user.Username
}

func timePtr(t time.Time) *time.Time { return &t }

func (s *AuthService) updateExistingUser(user *model.User, encryptedToken, name string, avatarURL *string) (*model.User, error) {
	user.MisskeyToken = encryptedToken
	user.Name = name
	if avatarURL != nil {
		user.AvatarURL = avatarURL
	}
	if err := s.userRepo.UpdateToken(user.ID, encryptedToken); err != nil {
		return nil, err
	}
	if err := s.userRepo.UpdateLastLogin(user.ID); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) createNewUser(miauthUser model.MiAuthUser, userHost, encryptedToken, name string) (*model.User, error) {
	hasAdmin, err := s.userRepo.HasAdmin()
	if err != nil {
		return nil, err
	}

	isAdmin := !hasAdmin && s.isHostAllowed(userHost)

	newUser := &model.User{
		MisskeyUserID:   miauthUser.ID,
		MisskeyUsername: miauthUser.Username,
		MisskeyHost:     userHost,
		MisskeyToken:    encryptedToken,
		Name:            name,
		AvatarURL:       miauthUser.AvatarURL,
		IsAdmin:         isAdmin,
		IsActive:        true,
		LastLoginAt:     timePtr(time.Now()),
	}
	if err := s.userRepo.Create(newUser); err != nil {
		return nil, err
	}
	if isAdmin {
		s.settingRepo.InitDefaults(newUser.ID)
	}
	return newUser, nil
}
