package repository

import (
	"database/sql"

	"miSchedule/internal/model"

	"github.com/google/uuid"
)

type InstanceRepo struct {
	db *sql.DB
}

func NewInstanceRepo(db *sql.DB) *InstanceRepo {
	return &InstanceRepo{db: db}
}

func (r *InstanceRepo) Create(id model.InstanceAllow) error {
	var createdBy interface{}
	if id.CreatedBy != uuid.Nil {
		createdBy = id.CreatedBy
	}
	return r.db.QueryRow(
		`INSERT INTO instance_allowlist (host, description, enabled, protected, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`,
		id.Host, id.Description, id.Enabled, id.Protected, createdBy,
	).Scan(&id.ID, &id.CreatedAt, &id.UpdatedAt)
}

func (r *InstanceRepo) List() ([]model.InstanceAllow, error) {
	rows, err := r.db.Query(
		`SELECT id, host, description, enabled, protected, created_by, created_at, updated_at
		FROM instance_allowlist ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instances []model.InstanceAllow
	for rows.Next() {
		var i model.InstanceAllow
		if err := rows.Scan(&i.ID, &i.Host, &i.Description, &i.Enabled,
			&i.Protected, &i.CreatedBy, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		instances = append(instances, i)
	}
	return instances, nil
}

func (r *InstanceRepo) FindByID(id uuid.UUID) (*model.InstanceAllow, error) {
	i := &model.InstanceAllow{}
	err := r.db.QueryRow(
		`SELECT id, host, description, enabled, protected, created_by, created_at, updated_at
		FROM instance_allowlist WHERE id = $1`, id,
	).Scan(&i.ID, &i.Host, &i.Description, &i.Enabled,
		&i.Protected, &i.CreatedBy, &i.CreatedAt, &i.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return i, err
}

func (r *InstanceRepo) FindByHost(host string) (*model.InstanceAllow, error) {
	i := &model.InstanceAllow{}
	err := r.db.QueryRow(
		`SELECT id, host, description, enabled, protected, created_by, created_at, updated_at
		FROM instance_allowlist WHERE host = $1`, host,
	).Scan(&i.ID, &i.Host, &i.Description, &i.Enabled,
		&i.Protected, &i.CreatedBy, &i.CreatedAt, &i.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return i, err
}

func (r *InstanceRepo) Update(i *model.InstanceAllow) error {
	_, err := r.db.Exec(
		`UPDATE instance_allowlist SET description=$1, enabled=$2, updated_at=NOW()
		WHERE id=$3`,
		i.Description, i.Enabled, i.ID,
	)
	return err
}

func (r *InstanceRepo) Delete(id uuid.UUID) error {
	_, err := r.db.Exec(`DELETE FROM instance_allowlist WHERE id = $1`, id)
	return err
}

func (r *InstanceRepo) IsHostAllowed(host string) (bool, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM instance_allowlist WHERE host = $1 AND enabled = true`, host,
	).Scan(&count)
	return count > 0, err
}

func (r *InstanceRepo) Seed(hosts []string, createdBy uuid.UUID) error {
	for _, host := range hosts {
		existing, _ := r.FindByHost(host)
		if existing == nil {
			r.Create(model.InstanceAllow{
				Host:        host,
				Enabled:     true,
				Protected:   true,
				Description: nil,
				CreatedBy:   createdBy,
			})
		}
	}
	return nil
}
