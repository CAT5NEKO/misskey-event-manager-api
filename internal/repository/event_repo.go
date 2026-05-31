package repository

import (
	"database/sql"
	"fmt"
	"time"

	"miSchedule/internal/model"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type EventRepo struct {
	db *sql.DB
}

func NewEventRepo(db *sql.DB) *EventRepo {
	return &EventRepo{db: db}
}

func (r *EventRepo) Create(event *model.Event) error {
	var notifTiming []int64
	for _, n := range event.NotificationTiming {
		notifTiming = append(notifTiming, int64(n))
	}

	return r.db.QueryRow(
		`INSERT INTO events (creator_id, title, description, location, notes, event_date, deadline, notification_timing, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`,
		event.CreatorID, event.Title, event.Description, event.Location, event.Notes,
		event.EventDate, event.Deadline, pq.Array(notifTiming), event.Status,
	).Scan(&event.ID, &event.CreatedAt, &event.UpdatedAt)
}

func (r *EventRepo) FindByID(id uuid.UUID) (*model.Event, error) {
	event := &model.Event{}
	var notifTiming []int64

	err := r.db.QueryRow(
		`SELECT id, creator_id, title, description, location, notes, event_date, deadline,
		 notification_timing, notified_at, status, created_at, updated_at
		FROM events WHERE id = $1`, id,
	).Scan(&event.ID, &event.CreatorID, &event.Title, &event.Description, &event.Location,
		&event.Notes, &event.EventDate, &event.Deadline, pq.Array(&notifTiming),
		&event.NotifiedAt, &event.Status, &event.CreatedAt, &event.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	for _, n := range notifTiming {
		event.NotificationTiming = append(event.NotificationTiming, int(n))
	}
	return event, nil
}

func (r *EventRepo) List(filterStatus, filter string, userID uuid.UUID, participating bool, page, limit int) ([]model.Event, int, error) {
	var total int
	var rows *sql.Rows
	var err error

	offset := (page - 1) * limit

	filterSQL := ""
	args := []interface{}{}
	if filter == "active" {
		filterSQL = " AND e.status = 'active' AND (e.deadline IS NULL OR e.deadline > NOW())"
	} else if filter == "past" {
		filterSQL = " AND (e.status IN ('completed', 'cancelled') OR (e.status = 'active' AND e.deadline IS NOT NULL AND e.deadline <= NOW()))"
	} else if filterStatus != "" {
		filterSQL = " AND e.status = $1"
		args = append(args, filterStatus)
	}

	if participating {
		countSQL := `SELECT COUNT(DISTINCT e.id) FROM events e
			INNER JOIN event_participants ep ON e.id = ep.event_id
			WHERE ep.user_id = $1` + filterSQL
		allArgs := append([]interface{}{userID}, args...)
		err = r.db.QueryRow(countSQL, allArgs...).Scan(&total)
		if err != nil {
			return nil, 0, err
		}

		querySQL := `SELECT DISTINCT e.id, e.creator_id, e.title, e.description, e.location, e.notes,
			 e.event_date, e.deadline, e.notification_timing, e.notified_at,
			 e.status, e.created_at, e.updated_at
			FROM events e
			INNER JOIN event_participants ep ON e.id = ep.event_id
			WHERE ep.user_id = $1` + filterSQL + `
			ORDER BY e.deadline ASC NULLS LAST, e.created_at DESC
			LIMIT $` + fmt.Sprintf("%d", len(allArgs)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(allArgs)+2)
		allArgs = append(allArgs, limit, offset)
		rows, err = r.db.Query(querySQL, allArgs...)
	} else {
		countSQL := `SELECT COUNT(*) FROM events e WHERE 1=1` + filterSQL
		err = r.db.QueryRow(countSQL, args...).Scan(&total)
		if err != nil {
			return nil, 0, err
		}

		querySQL := `SELECT id, creator_id, title, description, location, notes,
			 event_date, deadline, notification_timing, notified_at,
			 status, created_at, updated_at
			FROM events e WHERE 1=1` + filterSQL + `
			ORDER BY created_at DESC
			LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)
		allArgs := append(args, limit, offset)
		rows, err = r.db.Query(querySQL, allArgs...)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []model.Event
	for rows.Next() {
		var e model.Event
		var nt []int64
		if err := rows.Scan(&e.ID, &e.CreatorID, &e.Title, &e.Description, &e.Location,
			&e.Notes, &e.EventDate, &e.Deadline, pq.Array(&nt), &e.NotifiedAt,
			&e.Status, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, 0, err
		}
		for _, n := range nt {
			e.NotificationTiming = append(e.NotificationTiming, int(n))
		}
		events = append(events, e)
	}
	return events, total, nil
}

func (r *EventRepo) ListAll(page, limit int) ([]model.Event, int, error) {
	var total int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(
		`SELECT id, creator_id, title, description, location, notes,
		 event_date, deadline, notification_timing, notified_at,
		 status, created_at, updated_at
		FROM events ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []model.Event
	for rows.Next() {
		var e model.Event
		var nt []int64
		if err := rows.Scan(&e.ID, &e.CreatorID, &e.Title, &e.Description, &e.Location,
			&e.Notes, &e.EventDate, &e.Deadline, pq.Array(&nt), &e.NotifiedAt,
			&e.Status, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, 0, err
		}
		for _, n := range nt {
			e.NotificationTiming = append(e.NotificationTiming, int(n))
		}
		events = append(events, e)
	}
	return events, total, nil
}

func (r *EventRepo) Update(event *model.Event) error {
	var notifTiming []int64
	for _, n := range event.NotificationTiming {
		notifTiming = append(notifTiming, int64(n))
	}

	_, err := r.db.Exec(
		`UPDATE events SET title=$1, description=$2, location=$3, notes=$4,
		 event_date=$5, deadline=$6, notification_timing=$7, status=$8, updated_at=NOW()
		WHERE id=$9`,
		event.Title, event.Description, event.Location, event.Notes,
		event.EventDate, event.Deadline, pq.Array(notifTiming),
		event.Status, event.ID,
	)
	return err
}

func (r *EventRepo) Delete(id uuid.UUID) error {
	_, err := r.db.Exec(`DELETE FROM events WHERE id = $1`, id)
	return err
}

func (r *EventRepo) UpdateNotifiedAt(eventID uuid.UUID, timingIndex int, notifiedAt time.Time) error {
	var nt []int64
	var na model.TimeArray

	err := r.db.QueryRow(
		`SELECT notification_timing, notified_at FROM events WHERE id = $1`,
		eventID,
	).Scan(pq.Array(&nt), &na)
	if err != nil {
		return err
	}

	for len(na) <= timingIndex {
		na = append(na, time.Time{})
	}
	na[timingIndex] = notifiedAt

	_, err = r.db.Exec(
		`UPDATE events SET notified_at = $1, updated_at = NOW() WHERE id = $2`,
		na, eventID,
	)
	return err
}

func (r *EventRepo) FindEventsNeedingNotification() ([]model.Event, error) {
	rows, err := r.db.Query(
		`SELECT id, creator_id, title, description, location, notes,
		 event_date, deadline, notification_timing, notified_at, status, created_at, updated_at
		FROM events WHERE status = 'active' AND deadline IS NOT NULL AND notification_timing IS NOT NULL
		AND cardinality(notification_timing) > 0`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.Event
	for rows.Next() {
		var e model.Event
		var nt []int64
		if err := rows.Scan(&e.ID, &e.CreatorID, &e.Title, &e.Description, &e.Location,
			&e.Notes, &e.EventDate, &e.Deadline, pq.Array(&nt), &e.NotifiedAt,
			&e.Status, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		for _, n := range nt {
			e.NotificationTiming = append(e.NotificationTiming, int(n))
		}
		events = append(events, e)
	}
	return events, nil
}

func (r *EventRepo) GetCreatorID(eventID uuid.UUID) (uuid.UUID, error) {
	var creatorID uuid.UUID
	err := r.db.QueryRow(`SELECT creator_id FROM events WHERE id = $1`, eventID).Scan(&creatorID)
	return creatorID, err
}

func scanParticipants(rows *sql.Rows) ([]model.Participant, error) {
	var participants []model.Participant
	for rows.Next() {
		var p model.Participant
		if err := rows.Scan(&p.ID, &p.EventID, &p.UserID, &p.Status, &p.Comment,
			&p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}
	return participants, nil
}
