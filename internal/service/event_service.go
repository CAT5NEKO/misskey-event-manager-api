package service

import (
	"encoding/json"
	"fmt"
	"log"

	"miSchedule/internal/model"
	"miSchedule/internal/repository"

	"github.com/google/uuid"
)

type EventService struct {
	eventRepo       *repository.EventRepo
	participantRepo *repository.ParticipantRepo
	userRepo        *repository.UserRepo
	auditRepo       *repository.AuditLogRepo
	settingRepo     *repository.SettingRepo
}

func NewEventService(
	eventRepo *repository.EventRepo,
	participantRepo *repository.ParticipantRepo,
	userRepo *repository.UserRepo,
	auditRepo *repository.AuditLogRepo,
	settingRepo *repository.SettingRepo,
) *EventService {
	return &EventService{
		eventRepo: eventRepo, participantRepo: participantRepo,
		userRepo: userRepo, auditRepo: auditRepo, settingRepo: settingRepo,
	}
}

func (s *EventService) Create(input model.CreateEventInput, creatorID uuid.UUID, ipAddress, userAgent string) (*model.Event, error) {
	if len(input.NotificationTiming) == 0 {
		setting, _ := s.settingRepo.Get("notification.default_timing")
		if setting != nil && len(setting.Value) > 0 {
			var defaults []int
			if err := json.Unmarshal(setting.Value, &defaults); err == nil && len(defaults) > 0 {
				input.NotificationTiming = defaults
			} else {
				input.NotificationTiming = []int{1440, 180}
			}
		} else {
			input.NotificationTiming = []int{1440, 180}
		}
	}

	event := &model.Event{
		CreatorID:          creatorID,
		Title:              input.Title,
		Description:        input.Description,
		Location:           input.Location,
		Notes:              input.Notes,
		EventDate:          input.EventDate,
		Deadline:           input.Deadline,
		NotificationTiming: input.NotificationTiming,
		Status:             model.EventStatusActive,
	}

	if err := s.eventRepo.Create(event); err != nil {
		return nil, err
	}

	s.participantRepo.Create(&model.Participant{
		EventID: event.ID,
		UserID:  creatorID,
		Status:  model.ParticipantStatusAttending,
	})

	s.logAudit(creatorID, "event.create", "event", &event.ID, ipAddress, userAgent, nil)
	return event, nil
}

func (s *EventService) GetByID(eventID uuid.UUID, currentUserID uuid.UUID) (*model.Event, error) {
	event, err := s.eventRepo.FindByID(eventID)
	if err != nil || event == nil {
		return nil, fmt.Errorf("イベントが見つかりません")
	}

	creator, _ := s.userRepo.FindByID(event.CreatorID)
	if creator != nil {
		event.Creator = creator
	}

	participants, _ := s.participantRepo.ListByEventWithUsers(eventID)
	for i, p := range participants {
		u, _ := s.userRepo.FindByID(p.UserID)
		if u != nil {
			participants[i].User = u
		}
		if p.UserID == currentUserID {
			status := p.Status
			event.CurrentUserStatus = &status
		}
	}
	event.Participants = participants

	return event, nil
}

func (s *EventService) List(filterStatus, filter string, userID uuid.UUID, participating bool, page, limit int) (*model.PaginatedEvents, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	events, total, err := s.eventRepo.List(filterStatus, filter, userID, participating, page, limit)
	if err != nil {
		return nil, err
	}

	for i := range events {
		creator, _ := s.userRepo.FindByID(events[i].CreatorID)
		if creator != nil {
			events[i].Creator = creator
		}
	}

	return &model.PaginatedEvents{
		Events:     events,
		TotalCount: total,
		Page:       page,
		Limit:      limit,
	}, nil
}

func (s *EventService) Update(eventID uuid.UUID, input model.UpdateEventInput, userID uuid.UUID, isAdmin bool, ipAddress, userAgent string) (*model.Event, error) {
	event, err := s.eventRepo.FindByID(eventID)
	if err != nil || event == nil {
		return nil, fmt.Errorf("イベントが見つかりません")
	}

	if event.CreatorID != userID && !isAdmin {
		return nil, fmt.Errorf("権限がありません")
	}

	if input.Title != nil {
		event.Title = *input.Title
	}
	if input.Description != nil {
		event.Description = input.Description
	}
	if input.Location != nil {
		event.Location = input.Location
	}
	if input.Notes != nil {
		event.Notes = input.Notes
	}
	if input.EventDate != nil {
		event.EventDate = input.EventDate
	}
	if input.Deadline != nil {
		event.Deadline = input.Deadline
	}
	if input.NotificationTiming != nil {
		event.NotificationTiming = *input.NotificationTiming
		event.NotifiedAt = nil
	}
	if input.Status != nil {
		event.Status = *input.Status
	}

	if err := s.eventRepo.Update(event); err != nil {
		return nil, err
	}

	s.logAudit(userID, "event.update", "event", &eventID, ipAddress, userAgent, nil)
	return event, nil
}

func (s *EventService) Delete(eventID uuid.UUID, userID uuid.UUID, isAdmin bool, ipAddress, userAgent string) error {
	event, err := s.eventRepo.FindByID(eventID)
	if err != nil || event == nil {
		return fmt.Errorf("イベントが見つかりません")
	}

	if event.CreatorID != userID && !isAdmin {
		return fmt.Errorf("権限がありません")
	}

	if err := s.eventRepo.Delete(eventID); err != nil {
		return err
	}

	s.logAudit(userID, "event.delete", "event", &eventID, ipAddress, userAgent, nil)
	return nil
}

func (s *EventService) Join(eventID uuid.UUID, userID uuid.UUID, input model.JoinEventInput, ipAddress, userAgent string) (*model.Participant, error) {
	event, err := s.eventRepo.FindByID(eventID)
	if err != nil || event == nil {
		return nil, fmt.Errorf("イベントが見つかりません")
	}
	if event.Status != model.EventStatusActive {
		return nil, fmt.Errorf("このイベントは終了しています")
	}

	existing, err := s.participantRepo.FindByEventAndUser(eventID, userID)
	if err != nil {
		return nil, err
	}

	status := input.Status
	if status == "" {
		status = model.ParticipantStatusAttending
	}

	if existing != nil {
		if err := s.participantRepo.UpdateStatus(existing.ID, status, input.Comment); err != nil {
			return nil, err
		}
		existing.Status = status
		s.logAudit(userID, "participant.update", "event", &eventID, ipAddress, userAgent, nil)
		return existing, nil
	}

	p := &model.Participant{
		EventID: eventID,
		UserID:  userID,
		Status:  status,
		Comment: input.Comment,
	}
	if err := s.participantRepo.Create(p); err != nil {
		return nil, err
	}

	s.logAudit(userID, "participant.join", "event", &eventID, ipAddress, userAgent, nil)
	return p, nil
}

func (s *EventService) Leave(eventID uuid.UUID, userID uuid.UUID, ipAddress, userAgent string) error {
	existing, err := s.participantRepo.FindByEventAndUser(eventID, userID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("参加していません")
	}

	s.logAudit(userID, "participant.leave", "event", &eventID, ipAddress, userAgent, nil)
	return s.participantRepo.Delete(existing.ID)
}

func (s *EventService) logAudit(actorID uuid.UUID, action, targetType string, targetID *uuid.UUID, ipAddress, userAgent string, details map[string]interface{}) {
	var detailsBytes []byte
	if details != nil {
		detailsBytes, _ = json.Marshal(details)
	}
	if err := s.auditRepo.Create(&model.AuditLog{
		ActorID:    actorID,
		Action:     action,
		TargetType: &targetType,
		TargetID:   targetID,
		Details:    json.RawMessage(detailsBytes),
		IPAddress:  &ipAddress,
		UserAgent:  &userAgent,
	}); err != nil {
		log.Printf("audit log error: %v", err)
	}
}
