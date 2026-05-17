package model

const (
	EventStatusActive    = "active"
	EventStatusCancelled = "cancelled"
	EventStatusCompleted = "completed"

	ParticipantStatusAttending = "attending"
	ParticipantStatusDeclined  = "declined"
	ParticipantStatusPending   = "pending"

	RevokedByUser   = "user"
	RevokedByAdmin  = "admin"
	RevokedBySystem = "system"
)
