package repository

import (
	"database/sql"

	"miSchedule/internal/auth"
	"miSchedule/internal/model"

	"github.com/google/uuid"
)

type UserRepo struct {
	db    *sql.DB
	crypto *auth.TokenCrypto
}

func NewUserRepo(db *sql.DB, crypto *auth.TokenCrypto) *UserRepo {
	return &UserRepo{db: db, crypto: crypto}
}

func (r *UserRepo) Create(user *model.User) error {
	return r.db.QueryRow(
		`INSERT INTO users (misskey_user_id, misskey_username, misskey_host, misskey_token, name, avatar_url, is_admin, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`,
		user.MisskeyUserID, user.MisskeyUsername, user.MisskeyHost,
		user.MisskeyToken, user.Name, user.AvatarURL,
		user.IsAdmin, user.IsActive,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *UserRepo) FindByMisskeyID(misskeyUserID, misskeyHost string) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRow(
		`SELECT id, misskey_user_id, misskey_username, misskey_host, misskey_token, name, avatar_url, is_admin, is_active, last_login_at, created_at, updated_at
		FROM users WHERE misskey_user_id = $1 AND misskey_host = $2`,
		misskeyUserID, misskeyHost,
	).Scan(&user.ID, &user.MisskeyUserID, &user.MisskeyUsername, &user.MisskeyHost,
		&user.MisskeyToken, &user.Name, &user.AvatarURL,
		&user.IsAdmin, &user.IsActive, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepo) FindByID(id uuid.UUID) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRow(
		`SELECT id, misskey_user_id, misskey_username, misskey_host, misskey_token, name, avatar_url, is_admin, is_active, last_login_at, created_at, updated_at
		FROM users WHERE id = $1`, id,
	).Scan(&user.ID, &user.MisskeyUserID, &user.MisskeyUsername, &user.MisskeyHost,
		&user.MisskeyToken, &user.Name, &user.AvatarURL,
		&user.IsAdmin, &user.IsActive, &user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepo) DecryptToken(user *model.User) (string, error) {
	return r.crypto.Decrypt(user.MisskeyToken)
}

func (r *UserRepo) UpdateToken(userID uuid.UUID, encryptedToken string) error {
	_, err := r.db.Exec(
		`UPDATE users SET misskey_token = $1, updated_at = NOW() WHERE id = $2`,
		encryptedToken, userID,
	)
	return err
}

func (r *UserRepo) UpdateLastLogin(userID uuid.UUID) error {
	_, err := r.db.Exec(`UPDATE users SET last_login_at = NOW() WHERE id = $1`, userID)
	return err
}

func (r *UserRepo) Deactivate(userID uuid.UUID) error {
	_, err := r.db.Exec(
		`UPDATE users SET is_active = false, misskey_token = '', updated_at = NOW() WHERE id = $1`,
		userID,
	)
	return err
}

func (r *UserRepo) Delete(userID uuid.UUID) error {
	_, err := r.db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	return err
}

func (r *UserRepo) List(page, limit int, search string) ([]model.User, int, error) {
	var total int
	searchPattern := "%" + search + "%"
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM users WHERE misskey_username ILIKE $1 OR name ILIKE $1 OR misskey_host ILIKE $1`,
		searchPattern,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.Query(
		`SELECT id, misskey_user_id, misskey_username, misskey_host, misskey_token, name, avatar_url, is_admin, is_active, last_login_at, created_at, updated_at
		FROM users WHERE misskey_username ILIKE $1 OR name ILIKE $1 OR misskey_host ILIKE $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		searchPattern, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.MisskeyUserID, &u.MisskeyUsername, &u.MisskeyHost,
			&u.MisskeyToken, &u.Name, &u.AvatarURL,
			&u.IsAdmin, &u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, nil
}

func (r *UserRepo) HasAdmin() (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users WHERE is_admin = true AND is_active = true`).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
