package repository

import (
	"database/sql"

	"miSchedule/internal/model"

	"github.com/google/uuid"
)

type ParticipantRepo struct {
	db *sql.DB
}

func NewParticipantRepo(db *sql.DB) *ParticipantRepo {
	return &ParticipantRepo{db: db}
}

func (r *ParticipantRepo) Create(p *model.Participant) error {
	return r.db.QueryRow(
		`INSERT INTO event_participants (event_id, user_id, status, comment)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`,
		p.EventID, p.UserID, p.Status, p.Comment,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *ParticipantRepo) FindByEventAndUser(eventID, userID uuid.UUID) (*model.Participant, error) {
	p := &model.Participant{}
	err := r.db.QueryRow(
		`SELECT id, event_id, user_id, status, comment, created_at, updated_at
		FROM event_participants WHERE event_id = $1 AND user_id = $2`,
		eventID, userID,
	).Scan(&p.ID, &p.EventID, &p.UserID, &p.Status, &p.Comment, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *ParticipantRepo) UpdateStatus(id uuid.UUID, status string, comment *string) error {
	_, err := r.db.Exec(
		`UPDATE event_participants SET status = $1, comment = COALESCE($2, comment), updated_at = NOW()
		WHERE id = $3`,
		status, comment, id,
	)
	return err
}

func (r *ParticipantRepo) Delete(id uuid.UUID) error {
	_, err := r.db.Exec(`DELETE FROM event_participants WHERE id = $1`, id)
	return err
}

func (r *ParticipantRepo) ListByEvent(eventID uuid.UUID) ([]model.Participant, error) {
	rows, err := r.db.Query(
		`SELECT id, event_id, user_id, status, comment, created_at, updated_at
		FROM event_participants WHERE event_id = $1
		ORDER BY created_at ASC`,
		eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanParticipants(rows)
}

func (r *ParticipantRepo) ListByEventWithUsers(eventID uuid.UUID) ([]model.Participant, error) {
	rows, err := r.db.Query(
		`SELECT ep.id, ep.event_id, ep.user_id, ep.status, ep.comment, ep.created_at, ep.updated_at
		FROM event_participants ep
		WHERE ep.event_id = $1
		ORDER BY ep.created_at ASC`,
		eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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
