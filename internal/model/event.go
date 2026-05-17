package model

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID                 uuid.UUID     `json:"id"`
	CreatorID          uuid.UUID     `json:"creator_id"`
	Title              string        `json:"title"`
	Description        *string       `json:"description"`
	Location           *string       `json:"location"`
	Notes              *string       `json:"notes"`
	EventDate          *time.Time    `json:"event_date"`
	Deadline           *time.Time    `json:"deadline"`
	NotificationTiming []int         `json:"notification_timing"`
	NotifiedAt         TimeArray     `json:"-"`
	Status             string        `json:"status"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
	Creator            *User         `json:"creator,omitempty"`
	Participants       []Participant `json:"participants,omitempty"`
	CurrentUserStatus  *string       `json:"current_user_status,omitempty"`
}

type Participant struct {
	ID        uuid.UUID `json:"id"`
	EventID   uuid.UUID `json:"event_id"`
	UserID    uuid.UUID `json:"user_id"`
	Status    string    `json:"status"`
	Comment   *string   `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      *User     `json:"user,omitempty"`
}

type CreateEventInput struct {
	Title              string     `json:"title"`
	Description        *string    `json:"description"`
	Location           *string    `json:"location"`
	Notes              *string    `json:"notes"`
	EventDate          *time.Time `json:"event_date"`
	Deadline           *time.Time `json:"deadline"`
	NotificationTiming []int      `json:"notification_timing"`
}

type UpdateEventInput struct {
	Title              *string    `json:"title"`
	Description        *string    `json:"description"`
	Location           *string    `json:"location"`
	Notes              *string    `json:"notes"`
	EventDate          *time.Time `json:"event_date"`
	Deadline           *time.Time `json:"deadline"`
	NotificationTiming *[]int     `json:"notification_timing"`
	Status             *string    `json:"status"`
}

type JoinEventInput struct {
	Status  string  `json:"status"`
	Comment *string `json:"comment"`
}

type PaginatedEvents struct {
	Events     []Event `json:"events"`
	TotalCount int     `json:"total_count"`
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
}
