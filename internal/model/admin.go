package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type InstanceAllow struct {
	ID          uuid.UUID `json:"id"`
	Host        string    `json:"host"`
	Description *string   `json:"description"`
	Enabled     bool      `json:"enabled"`
	Protected   bool      `json:"protected"`
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AuditLog struct {
	ID         uuid.UUID       `json:"id"`
	ActorID    uuid.UUID       `json:"actor_id"`
	Action     string          `json:"action"`
	TargetType *string         `json:"target_type"`
	TargetID   *uuid.UUID      `json:"target_id"`
	Details    json.RawMessage `json:"details"`
	IPAddress  *string         `json:"ip_address"`
	UserAgent  *string         `json:"user_agent"`
	CreatedAt  time.Time       `json:"created_at"`
	Actor      *User           `json:"actor,omitempty"`
}

type AuditLogListParams struct {
	ActorID    *uuid.UUID
	Action     *string
	TargetType *string
	From       *time.Time
	To         *time.Time
	Page       int
	Limit      int
}

type PaginatedAuditLogs struct {
	Logs       []AuditLog `json:"logs"`
	TotalCount int        `json:"total_count"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
}

type SystemSetting struct {
	ID        uuid.UUID       `json:"id"`
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	UpdatedBy uuid.UUID       `json:"updated_by"`
	UpdatedAt time.Time       `json:"updated_at"`
}
