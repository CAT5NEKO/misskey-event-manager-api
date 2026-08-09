package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"miSchedule/internal/config"
	"miSchedule/internal/model"
	"miSchedule/internal/repository"

	"github.com/google/uuid"
)

type NotificationScheduler struct {
	cfg             *config.Config
	eventRepo       *repository.EventRepo
	participantRepo *repository.ParticipantRepo
	userRepo        *repository.UserRepo
	auditRepo       *repository.AuditLogRepo
}

func NewNotificationScheduler(
	cfg *config.Config,
	eventRepo *repository.EventRepo,
	participantRepo *repository.ParticipantRepo,
	userRepo *repository.UserRepo,
	auditRepo *repository.AuditLogRepo,
) *NotificationScheduler {
	return &NotificationScheduler{
		cfg: cfg, eventRepo: eventRepo, participantRepo: participantRepo,
		userRepo: userRepo, auditRepo: auditRepo,
	}
}

func (ns *NotificationScheduler) Run() {
	interval := time.Duration(ns.cfg.NotificationInterval) * time.Second
	if interval < 30*time.Second {
		interval = 30 * time.Second
	}

	go func() {
		for {
			ns.cycle(interval)
		}
	}()
	log.Printf("notification scheduler started with %s interval", interval)
	if ns.cfg.PublicURL == "" {
		log.Printf("warning: PUBLIC_URL is not set; notifications will omit the event link")
	}
}

// cycle sends due notifications, then sleeps until the nearest future notification
// time (capped at interval so newly created or updated events are discovered in time).
// A single DB query feeds both steps.
func (ns *NotificationScheduler) cycle(interval time.Duration) {
	events, err := ns.eventRepo.FindEventsNeedingNotification()
	if err != nil {
		log.Printf("notification check error: %v", err)
		time.Sleep(interval)
		return
	}
	ns.notifyDue(events)
	time.Sleep(nextNotifyDelay(events, interval, time.Now()))
}

func (ns *NotificationScheduler) notifyDue(events []model.Event) {
	now := time.Now()
	for _, event := range events {
		if event.Deadline == nil {
			continue
		}

		for i, timingMin := range event.NotificationTiming {
			if len(event.NotifiedAt) > i && !event.NotifiedAt[i].IsZero() {
				continue
			}

			notifyAt := event.Deadline.Add(-time.Duration(timingMin) * time.Minute)
			if now.Before(notifyAt) {
				continue
			}

			ns.sendNotifications(event, i, timingMin)
		}
	}
}

func nextNotifyDelay(events []model.Event, interval time.Duration, now time.Time) time.Duration {
	next := interval
	for _, event := range events {
		if event.Deadline == nil {
			continue
		}
		for i, timingMin := range event.NotificationTiming {
			if len(event.NotifiedAt) > i && !event.NotifiedAt[i].IsZero() {
				continue
			}
			notifyAt := event.Deadline.Add(-time.Duration(timingMin) * time.Minute)
			if !now.Before(notifyAt) {
				continue
			}
			if d := notifyAt.Sub(now); d < next {
				next = d
			}
		}
	}
	if next < time.Second {
		next = time.Second
	}
	return next
}

func buildNotificationBody(title string, eventID uuid.UUID, timingMin int, eventURLBase string) string {
	var timingDesc string
	switch {
	case timingMin >= 1440:
		timingDesc = fmt.Sprintf("あと%d日", timingMin/1440)
	case timingMin >= 60:
		timingDesc = fmt.Sprintf("あと%d時間", timingMin/60)
	default:
		timingDesc = fmt.Sprintf("あと%d分", timingMin)
	}

	body := fmt.Sprintf("[期限] %s（%s）", title, timingDesc)
	if eventURLBase != "" {
		body += fmt.Sprintf("\n%s/events/%s", eventURLBase, eventID.String())
	}
	return body
}

func (ns *NotificationScheduler) sendNotifications(event model.Event, timingIndex, timingMin int) {
	notified := make(map[uuid.UUID]bool)

	creator, err := ns.userRepo.FindByID(event.CreatorID)
	if err == nil && creator != nil && creator.IsActive {
		token, err := ns.userRepo.DecryptToken(creator)
		if err == nil {
			ns.sendSingleNotification(creator.MisskeyHost, token, event.Title, event.ID, timingMin)
			notified[creator.ID] = true
		}
	}

	participants, err := ns.participantRepo.ListByEvent(event.ID)
	if err != nil {
		log.Printf("failed to get participants for event %s: %v", event.ID, err)
		return
	}

	for _, p := range participants {
		if p.Status == model.ParticipantStatusDeclined {
			continue
		}
		if notified[p.UserID] {
			continue
		}

		user, err := ns.userRepo.FindByID(p.UserID)
		if err != nil || user == nil || !user.IsActive {
			continue
		}

		token, err := ns.userRepo.DecryptToken(user)
		if err != nil {
			log.Printf("failed to decrypt token for user %s: %v", user.ID, err)
			ns.userRepo.Deactivate(user.ID)
			continue
		}

		ns.sendSingleNotification(user.MisskeyHost, token, event.Title, event.ID, timingMin)
		notified[user.ID] = true
	}

	if len(notified) > 0 {
		log.Printf("notification sent for event %s (timing %d min): %d recipients", event.Title, timingMin, len(notified))
	}
	ns.eventRepo.UpdateNotifiedAt(event.ID, timingIndex, time.Now())
}

func (ns *NotificationScheduler) sendSingleNotification(host, token, title string, eventID uuid.UUID, timingMin int) {
	body := buildNotificationBody(title, eventID, timingMin, ns.cfg.PublicURL)

	payload := map[string]interface{}{
		"body": body,
	}
	payloadBytes, _ := json.Marshal(payload)

	resolvedHost := ns.cfg.ResolveHost(host)
	scheme := "https://"
	if strings.Contains(resolvedHost, "localhost") || strings.Contains(resolvedHost, "docker.internal") || strings.Contains(resolvedHost, "127.0.0.1") {
		scheme = "http://"
	}
	var baseURL string
	if strings.HasPrefix(resolvedHost, "http") {
		baseURL = fmt.Sprintf("%s/api/notifications/create", resolvedHost)
	} else if len(resolvedHost) > 0 && resolvedHost[len(resolvedHost)-1] == '/' {
		baseURL = fmt.Sprintf("%s%sapi/notifications/create", scheme, resolvedHost)
	} else {
		baseURL = fmt.Sprintf("%s%s/api/notifications/create", scheme, resolvedHost)
	}

	req, err := http.NewRequest("POST", baseURL, bytes.NewReader(payloadBytes))
	if err != nil {
		log.Printf("notification request error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("notification send error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		log.Printf("notification returned status %d for event", resp.StatusCode)
	}
}
