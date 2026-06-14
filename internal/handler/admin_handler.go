package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"miSchedule/internal/middleware"
	"miSchedule/internal/model"
	"miSchedule/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AdminHandler struct {
	adminService *service.AdminService
}

func NewAdminHandler(adminService *service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	search := r.URL.Query().Get("search")
	if len(search) > 100 {
		search = search[:100]
	}

	result, err := h.adminService.ListUsers(page, limit, search)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "ユーザーIDが不正です")
		return
	}

	user, err := h.adminService.GetUser(userID)
	if err != nil || user == nil {
		respondError(w, http.StatusNotFound, "ユーザーが見つかりません")
		return
	}
	respondJSON(w, http.StatusOK, user)
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "ユーザーIDが不正です")
		return
	}

	adminID, _ := middleware.GetUserID(r)
	ipAddress := middleware.GetIPAddress(r)
	userAgent := r.UserAgent()

	if err := h.adminService.DeleteUser(userID, adminID, ipAddress, userAgent); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}

func (h *AdminHandler) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "ユーザーIDが不正です")
		return
	}

	adminID, _ := middleware.GetUserID(r)
	ipAddress := middleware.GetIPAddress(r)
	userAgent := r.UserAgent()

	if err := h.adminService.DeactivateUser(userID, adminID, ipAddress, userAgent); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "user deactivated"})
}

func (h *AdminHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	result, err := h.adminService.ListEvents(page, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (h *AdminHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	eventID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "イベントIDが不正です")
		return
	}

	adminID, _ := middleware.GetUserID(r)
	ipAddress := middleware.GetIPAddress(r)
	userAgent := r.UserAgent()

	if err := h.adminService.DeleteEvent(eventID, adminID, ipAddress, userAgent); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "event deleted"})
}

func (h *AdminHandler) ListInstances(w http.ResponseWriter, r *http.Request) {
	instances, err := h.adminService.ListInstances()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if instances == nil {
		instances = []model.InstanceAllow{}
	}
	respondJSON(w, http.StatusOK, instances)
}

func (h *AdminHandler) AddInstance(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Host        string `json:"host"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "リクエスト形式が不正です")
		return
	}

	adminID, _ := middleware.GetUserID(r)
	ipAddress := middleware.GetIPAddress(r)
	userAgent := r.UserAgent()

	inst, err := h.adminService.AddInstance(input.Host, input.Description, adminID, ipAddress, userAgent)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, inst)
}

func (h *AdminHandler) UpdateInstance(w http.ResponseWriter, r *http.Request) {
	instanceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "インスタンスIDが不正です")
		return
	}

	var input struct {
		Enabled     *bool   `json:"enabled"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "リクエスト形式が不正です")
		return
	}

	adminID, _ := middleware.GetUserID(r)
	ipAddress := middleware.GetIPAddress(r)
	userAgent := r.UserAgent()

	inst, err := h.adminService.UpdateInstance(instanceID, input.Enabled, input.Description, adminID, ipAddress, userAgent)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, inst)
}

func (h *AdminHandler) DeleteInstance(w http.ResponseWriter, r *http.Request) {
	instanceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "インスタンスIDが不正です")
		return
	}

	adminID, _ := middleware.GetUserID(r)
	ipAddress := middleware.GetIPAddress(r)
	userAgent := r.UserAgent()

	if err := h.adminService.DeleteInstance(instanceID, adminID, ipAddress, userAgent); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "instance deleted"})
}

func (h *AdminHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	params := model.AuditLogListParams{
		Page:  1,
		Limit: 50,
	}

	if v := r.URL.Query().Get("page"); v != "" {
		params.Page, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		params.Limit, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("actor_id"); v != "" {
		id, _ := uuid.Parse(v)
		params.ActorID = &id
	}
	if v := r.URL.Query().Get("action"); v != "" {
		params.Action = &v
	}
	if v := r.URL.Query().Get("target_type"); v != "" {
		params.TargetType = &v
	}
	if v := r.URL.Query().Get("from"); v != "" {
		t, _ := time.Parse(time.RFC3339, v)
		params.From = &t
	}
	if v := r.URL.Query().Get("to"); v != "" {
		t, _ := time.Parse(time.RFC3339, v)
		params.To = &t
	}

	result, err := h.adminService.ListAuditLogs(params)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (h *AdminHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.adminService.GetSettings()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, settings)
}

func (h *AdminHandler) UpdateSetting(w http.ResponseWriter, r *http.Request) {
	var input map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "リクエスト形式が不正です")
		return
	}

	adminID, _ := middleware.GetUserID(r)
	ipAddress := middleware.GetIPAddress(r)
	userAgent := r.UserAgent()

	if err := h.adminService.UpdateSettings(input, adminID, ipAddress, userAgent); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "settings updated"})
}
