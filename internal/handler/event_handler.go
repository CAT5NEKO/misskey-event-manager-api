package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"miSchedule/internal/middleware"
	"miSchedule/internal/model"
	"miSchedule/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type EventHandler struct {
	eventService *service.EventService
}

func NewEventHandler(eventService *service.EventService) *EventHandler {
	return &EventHandler{eventService: eventService}
}

func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r)
	status := r.URL.Query().Get("status")
	filter := r.URL.Query().Get("filter")
	participating := r.URL.Query().Get("participating") == "true"
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	result, err := h.eventService.List(status, filter, userID, participating, page, limit)
	if err != nil {
		log.Printf("event list error: %v", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r)
	var input model.CreateEventInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "リクエスト形式が不正です")
		return
	}
	if input.Title == "" {
		respondError(w, http.StatusBadRequest, "タイトルを入力してください")
		return
	}

	now := time.Now()
	if input.EventDate != nil && input.EventDate.Before(now) {
		respondError(w, http.StatusBadRequest, "予定日は現在より未来の日時を指定してください")
		return
	}
	if input.Deadline != nil && input.Deadline.Before(now) {
		respondError(w, http.StatusBadRequest, "回答期限は現在より未来の日時を指定してください")
		return
	}
	if input.EventDate != nil && input.Deadline != nil && input.Deadline.After(*input.EventDate) {
		respondError(w, http.StatusBadRequest, "回答期限は予定日より前に設定してください")
		return
	}

	ipAddress := middleware.GetIPAddress(r)
	userAgent := r.UserAgent()

	event, err := h.eventService.Create(input, userID, ipAddress, userAgent)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, event)
}

func (h *EventHandler) Get(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "イベントIDが不正です")
		return
	}

	userID, _ := middleware.GetUserID(r)

	event, err := h.eventService.GetByID(eventID, userID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, event)
}

func (h *EventHandler) Update(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "イベントIDが不正です")
		return
	}

	userID, _ := middleware.GetUserID(r)
	isAdmin := middleware.IsAdmin(r)

	var input model.UpdateEventInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "リクエスト形式が不正です")
		return
	}

	now := time.Now()
	if input.EventDate != nil && input.EventDate.Before(now) {
		respondError(w, http.StatusBadRequest, "予定日は現在より未来の日時を指定してください")
		return
	}
	if input.Deadline != nil {
		if input.Deadline.Before(now) {
			respondError(w, http.StatusBadRequest, "回答期限は現在より未来の日時を指定してください")
			return
		}
		if input.EventDate != nil && input.Deadline.After(*input.EventDate) {
			respondError(w, http.StatusBadRequest, "回答期限は予定日より前に設定してください")
			return
		}
	}

	ipAddress := middleware.GetIPAddress(r)
	userAgent := r.UserAgent()

	event, err := h.eventService.Update(eventID, input, userID, isAdmin, ipAddress, userAgent)
	if err != nil {
		switch err.Error() {
		case "event not found":
			respondError(w, http.StatusNotFound, err.Error())
		case "permission denied":
			respondError(w, http.StatusForbidden, err.Error())
		default:
			respondError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondJSON(w, http.StatusOK, event)
}

func (h *EventHandler) Delete(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "イベントIDが不正です")
		return
	}

	userID, _ := middleware.GetUserID(r)
	isAdmin := middleware.IsAdmin(r)
	ipAddress := middleware.GetIPAddress(r)
	userAgent := r.UserAgent()

	if err := h.eventService.Delete(eventID, userID, isAdmin, ipAddress, userAgent); err != nil {
		switch err.Error() {
		case "event not found":
			respondError(w, http.StatusNotFound, err.Error())
		case "permission denied":
			respondError(w, http.StatusForbidden, err.Error())
		default:
			respondError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "event deleted"})
}

func (h *EventHandler) Join(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "イベントIDが不正です")
		return
	}

	userID, _ := middleware.GetUserID(r)

	var input model.JoinEventInput
	json.NewDecoder(r.Body).Decode(&input)

	ipAddress := middleware.GetIPAddress(r)
	userAgent := r.UserAgent()

	p, err := h.eventService.Join(eventID, userID, input, ipAddress, userAgent)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, p)
}

func (h *EventHandler) Leave(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "イベントIDが不正です")
		return
	}

	userID, _ := middleware.GetUserID(r)
	ipAddress := middleware.GetIPAddress(r)
	userAgent := r.UserAgent()

	if err := h.eventService.Leave(eventID, userID, ipAddress, userAgent); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "left event"})
}
