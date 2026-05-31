package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type RefreshTokenRepo struct {
	db *sql.DB
}

func NewRefreshTokenRepo(db *sql.DB) *RefreshTokenRepo {
	return &RefreshTokenRepo{db: db}
}

func (r *RefreshTokenRepo) Create(userID uuid.UUID, tokenHash, familyID string, expiresAt time.Time) error {
	_, err := r.db.Exec(
		`INSERT INTO refresh_tokens (user_id, token_hash, family_id, expires_at)
		VALUES ($1, $2, $3, $4)`,
		userID, tokenHash, familyID, expiresAt,
	)
	return err
}

func (r *RefreshTokenRepo) FindByHash(tokenHash string) (userID uuid.UUID, familyID string, revoked bool, err error) {
	err = r.db.QueryRow(
		`SELECT user_id, family_id, revoked FROM refresh_tokens WHERE token_hash = $1`, tokenHash,
	).Scan(&userID, &familyID, &revoked)
	if err == sql.ErrNoRows {
		return uuid.Nil, "", false, nil
	}
	return userID, familyID, revoked, err
}

func (r *RefreshTokenRepo) RevokeAllInFamily(familyID string) error {
	_, err := r.db.Exec(`UPDATE refresh_tokens SET revoked = true WHERE family_id = $1`, familyID)
	return err
}

func (r *RefreshTokenRepo) RevokeToken(tokenHash string) error {
	_, err := r.db.Exec(`UPDATE refresh_tokens SET revoked = true WHERE token_hash = $1`, tokenHash)
	return err
}

func (r *RefreshTokenRepo) RevokeAllForUser(userID uuid.UUID) error {
	_, err := r.db.Exec(`UPDATE refresh_tokens SET revoked = true WHERE user_id = $1`, userID)
	return err
}

func (r *RefreshTokenRepo) CleanExpired() error {
	_, err := r.db.Exec(`DELETE FROM refresh_tokens WHERE expires_at < NOW()`)
	return err
}
