package sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/zeldojov/gopost/internal/storage"
)

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{
		db: db,
	}
}

func (r *SessionRepository) CreateSession(
	sess storage.Session,
) error {
	data, err := json.Marshal(sess.Data)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(`
		INSERT INTO sessions (
			id,
			data,
			ip,
			user_agent,
			created_at,
			expires_at
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		sess.ID,
		data,
		sess.IP,
		sess.UserAgent,
		sess.CreatedAt,
		sess.ExpiresAt,
	)

	return err
}

func (r *SessionRepository) GetSession(id string) (storage.Session, error) {
	var (
		dataJSON  string
		ip        string
		userAgent string
		createdAt time.Time
		expiresAt time.Time
	)

	err := r.db.QueryRow(`
		SELECT data, ip, user_agent, created_at, expires_at
		FROM sessions
		WHERE id = ?
	`, id).Scan(
		&dataJSON,
		&ip,
		&userAgent,
		&createdAt,
		&expiresAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.Session{}, storage.ErrNotFound
		}

		return storage.Session{}, err
	}

	var data map[string]string

	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return storage.Session{}, err
	}

	return storage.Session{
		ID:        id,
		Data:      data,
		IP:        ip,
		UserAgent: userAgent,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}, nil
}

func (r *SessionRepository) UpdateSession(
	sess storage.Session,
) error {
	data, err := json.Marshal(sess.Data)
	if err != nil {
		return err
	}

	result, err := r.db.Exec(`
		UPDATE sessions
		SET
			data = ?,
			ip = ?,
			user_agent = ?,
			created_at = ?,
			expires_at = ?
		WHERE id = ?
	`,
		data,
		sess.IP,
		sess.UserAgent,
		sess.CreatedAt,
		sess.ExpiresAt,
		sess.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return storage.ErrNotFound
	}

	return nil
}

func (r *SessionRepository) DeleteSession(id string) error {
	result, err := r.db.Exec(`
		DELETE FROM sessions
		WHERE id = ?
	`, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return storage.ErrNotFound
	}

	return nil
}

func (r *SessionRepository) RegenerateSession(oldID string, sess storage.Session) error {
	data, err := json.Marshal(sess.Data)
	if err != nil {
		return err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO sessions (
			id,
			data,
			ip,
			user_agent,
			created_at,
			expires_at
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		sess.ID,
		data,
		sess.IP,
		sess.UserAgent,
		sess.CreatedAt,
		sess.ExpiresAt,
	)
	if err != nil {
		return err
	}

	result, err := tx.Exec(`
		DELETE FROM sessions
		WHERE id = ?
	`, oldID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return storage.ErrNotFound
	}

	return tx.Commit()
}

var _ storage.SessionRepository = (*SessionRepository)(nil)
