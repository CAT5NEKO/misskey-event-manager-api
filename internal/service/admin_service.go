package service

import (
	"encoding/json"
	"fmt"
	"time"

	"miSchedule/internal/config"
	"miSchedule/internal/model"
	"miSchedule/internal/repository"

	"github.com/google/uuid"
)

type AdminService struct {
	cfg          *config.Config
	userRepo     *repository.UserRepo
	eventRepo    *repository.EventRepo
	instanceRepo *repository.InstanceRepo
	auditRepo    *repository.AuditLogRepo
	settingRepo  *repository.SettingRepo
	refreshRepo  *repository.RefreshTokenRepo
}

func NewAdminService(
	cfg *config.Config,
	userRepo *repository.UserRepo,
	eventRepo *repository.EventRepo,
	instanceRepo *repository.InstanceRepo,
	auditRepo *repository.AuditLogRepo,
	settingRepo *repository.SettingRepo,
	refreshRepo *repository.RefreshTokenRepo,
) *AdminService {
	return &AdminService{
		cfg: cfg, userRepo: userRepo, eventRepo: eventRepo,
		instanceRepo: instanceRepo, auditRepo: auditRepo,
		settingRepo: settingRepo, refreshRepo: refreshRepo,
	}
}

func (s *AdminService) ListUsers(page, limit int, search string) (*model.PaginatedUsers, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	users, total, err := s.userRepo.List(page, limit, search)
	if err != nil {
		return nil, err
	}
	return &model.PaginatedUsers{Users: users, TotalCount: total, Page: page, Limit: limit}, nil
}

func (s *AdminService) GetUser(userID uuid.UUID) (*model.User, error) {
	return s.userRepo.FindByID(userID)
}

func (s *AdminService) DeleteUser(userID, adminID uuid.UUID, ipAddress, userAgent string) error {
	if userID == adminID {
		return fmt.Errorf("自分のアカウントは削除できません")
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return fmt.Errorf("ユーザーが見つかりません")
	}
	if user.IsAdmin {
		return fmt.Errorf("管理者アカウントは削除できません")
	}

	s.refreshRepo.RevokeAllForUser(userID)

	if err := s.userRepo.Delete(userID); err != nil {
		return err
	}

	s.logAudit(adminID, "admin.delete_user", "user", &userID, ipAddress, userAgent, nil)
	return nil
}

func (s *AdminService) DeactivateUser(userID, adminID uuid.UUID, ipAddress, userAgent string) error {
	if userID == adminID {
		return fmt.Errorf("自分のアカウントは無効化できません")
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return fmt.Errorf("ユーザーが見つかりません")
	}
	if user.IsAdmin {
		return fmt.Errorf("管理者アカウントは無効化できません")
	}

	if err := s.userRepo.Deactivate(userID); err != nil {
		return err
	}

	s.refreshRepo.RevokeAllForUser(userID)
	s.logAudit(adminID, "admin.deactivate_user", "user", &userID, ipAddress, userAgent, nil)
	return nil
}

func (s *AdminService) ListEvents(page, limit int) (*model.PaginatedEvents, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	events, total, err := s.eventRepo.ListAll(page, limit)
	if err != nil {
		return nil, err
	}
	for i := range events {
		creator, _ := s.userRepo.FindByID(events[i].CreatorID)
		if creator != nil {
			events[i].Creator = creator
		}
	}
	return &model.PaginatedEvents{Events: events, TotalCount: total, Page: page, Limit: limit}, nil
}

func (s *AdminService) DeleteEvent(eventID, adminID uuid.UUID, ipAddress, userAgent string) error {
	if err := s.eventRepo.Delete(eventID); err != nil {
		return err
	}
	s.logAudit(adminID, "admin.delete_event", "event", &eventID, ipAddress, userAgent, nil)
	return nil
}

func (s *AdminService) ListInstances() ([]model.InstanceAllow, error) {
	return s.instanceRepo.List()
}

func (s *AdminService) AddInstance(host, description string, adminID uuid.UUID, ipAddress, userAgent string) (*model.InstanceAllow, error) {
	inst := model.InstanceAllow{
		Host:        host,
		Description: &description,
		Enabled:     true,
		CreatedBy:   adminID,
	}
	if err := s.instanceRepo.Create(inst); err != nil {
		return nil, err
	}
	s.logAudit(adminID, "instance.add", "instance", &inst.ID, ipAddress, userAgent, nil)
	return &inst, nil
}

func (s *AdminService) UpdateInstance(instanceID uuid.UUID, enabled *bool, description *string, adminID uuid.UUID, ipAddress, userAgent string) (*model.InstanceAllow, error) {
	inst, err := s.instanceRepo.FindByID(instanceID)
	if err != nil || inst == nil {
		return nil, fmt.Errorf("インスタンスが見つかりません")
	}
	if enabled != nil {
		inst.Enabled = *enabled
	}
	if description != nil {
		inst.Description = description
	}
	if err := s.instanceRepo.Update(inst); err != nil {
		return nil, err
	}
	s.logAudit(adminID, "instance.update", "instance", &instanceID, ipAddress, userAgent, nil)
	return inst, nil
}

func (s *AdminService) DeleteInstance(instanceID, adminID uuid.UUID, ipAddress, userAgent string) error {
	inst, err := s.instanceRepo.FindByID(instanceID)
	if err != nil || inst == nil {
		return fmt.Errorf("インスタンスが見つかりません")
	}
	if inst.Protected {
		return fmt.Errorf("保護されたインスタンスは削除できません")
	}
	if err := s.instanceRepo.Delete(instanceID); err != nil {
		return err
	}
	s.logAudit(adminID, "instance.delete", "instance", &instanceID, ipAddress, userAgent, nil)
	return nil
}

func (s *AdminService) ListAuditLogs(params model.AuditLogListParams) (*model.PaginatedAuditLogs, error) {
	return s.auditRepo.List(params)
}

func (s *AdminService) GetSettings() (map[string]json.RawMessage, error) {
	settings, err := s.settingRepo.GetAll()
	if err != nil {
		return nil, err
	}
	result := make(map[string]json.RawMessage)
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result, nil
}

func (s *AdminService) UpdateSetting(key string, value json.RawMessage, adminID uuid.UUID, ipAddress, userAgent string) error {
	if err := s.settingRepo.Upsert(key, value, adminID); err != nil {
		return err
	}
	s.logAudit(adminID, "setting.update", "setting", nil, ipAddress, userAgent, map[string]interface{}{"key": key})
	return nil
}

func (s *AdminService) UpdateSettings(input map[string]json.RawMessage, adminID uuid.UUID, ipAddress, userAgent string) error {
	keys := make([]string, 0, len(input))
	for key, value := range input {
		if err := s.settingRepo.Upsert(key, value, adminID); err != nil {
			return err
		}
		keys = append(keys, key)
	}
	s.logAudit(adminID, "setting.update", "setting", nil, ipAddress, userAgent, map[string]interface{}{"keys": keys})
	return nil
}

func (s *AdminService) CleanupAuditLogs() error {
	cutoff := time.Now().AddDate(0, 0, -90)
	return s.auditRepo.DeleteOlderThan(cutoff)
}

func (s *AdminService) SeedInstances() {
	go func() {
		for _, host := range s.cfg.AllowedInstances {
			existing, _ := s.instanceRepo.FindByHost(host)
			if existing == nil {
				s.instanceRepo.Create(model.InstanceAllow{
					Host:      host,
					Enabled:   true,
					Protected: true,
				})
			}
		}
	}()
}

func (s *AdminService) logAudit(actorID uuid.UUID, action, targetType string, targetID *uuid.UUID, ipAddress, userAgent string, details map[string]interface{}) {
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
