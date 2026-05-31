package repository

import (
	"database/sql"
	"encoding/json"

	"miSchedule/internal/model"

	"github.com/google/uuid"
)

type SettingRepo struct {
	db *sql.DB
}

func NewSettingRepo(db *sql.DB) *SettingRepo {
	return &SettingRepo{db: db}
}

func (r *SettingRepo) Get(key string) (*model.SystemSetting, error) {
	s := &model.SystemSetting{}
	err := r.db.QueryRow(
		`SELECT id, key, value, updated_by, updated_at FROM system_settings WHERE key = $1`, key,
	).Scan(&s.ID, &s.Key, &s.Value, &s.UpdatedBy, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return s, err
}

func (r *SettingRepo) GetAll() ([]model.SystemSetting, error) {
	rows, err := r.db.Query(
		`SELECT id, key, value, updated_by, updated_at FROM system_settings ORDER BY key ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []model.SystemSetting
	for rows.Next() {
		var s model.SystemSetting
		if err := rows.Scan(&s.ID, &s.Key, &s.Value, &s.UpdatedBy, &s.UpdatedAt); err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}
	return settings, nil
}

func (r *SettingRepo) Upsert(key string, value json.RawMessage, updatedBy uuid.UUID) error {
	_, err := r.db.Exec(
		`INSERT INTO system_settings (key, value, updated_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE SET value = $2, updated_by = $3, updated_at = NOW()`,
		key, value, updatedBy,
	)
	return err
}

func (r *SettingRepo) InitDefaults(adminID uuid.UUID) error {
	defaults := map[string]string{
		"app.name":                     `"miSchedule"`,
		"notification.default_timing": `[1440, 180]`,
		"events.max_per_user":          `0`,
		"users.list_visible":           `true`,
		"users.allow_self_delete":      `true`,
	}
	for key, val := range defaults {
		existing, _ := r.Get(key)
		if existing == nil {
			if err := r.Upsert(key, json.RawMessage(val), adminID); err != nil {
				return err
			}
		}
	}
	return nil
}
