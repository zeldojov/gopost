package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/zeldojov/gopost/internal/session"
)

const (
	createSessionsTableQuery = `
CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	csrf_token TEXT NOT NULL,
	user_id TEXT NULL,
	user_ip TEXT NOT NULL,
	user_agent TEXT NOT NULL,
	user_data TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	expires_at DATETIME NOT NULL
);
`
	saveSessionQuery = `
INSERT INTO sessions (
	id,
	csrf_token,
	user_id,
	user_ip,
	user_agent,
	user_data,
	created_at,
	expires_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	csrf_token = excluded.csrf_token,
	user_id = excluded.user_id,
	user_ip = excluded.user_ip,
	user_agent = excluded.user_agent,
	user_data = excluded.user_data,
	expires_at = excluded.expires_at
`

	getSessionByIDQuery = `
SELECT
	csrf_token,
	user_id,
	user_ip,
	user_agent,
	user_data,
	created_at,
	expires_at
FROM sessions
WHERE id = ?
`

	deleteSessionQuery = `
DELETE FROM sessions
WHERE id = ?
`
)

func (s *Store) CreateSessionsTable() error {
	_, err := s.db.Exec(createSessionsTableQuery)
	return err
}

func (s *Store) SaveSession(sess *session.Session) error {
	data, err := json.Marshal(sess.UserData())
	if err != nil {
		return err
	}

	_, err = s.db.Exec(
		saveSessionQuery,
		sess.ID(),
		sess.CSRFToken(),
		sess.UserID(),
		sess.UserIP(),
		sess.UserAgent(),
		data,
		sess.CreatedAt(),
		sess.ExpiresAt(),
	)

	return err
}

func (s *Store) LoadSession(sess *session.Session, id string) error {
	var (
		csrfToken string
		userID    sql.NullString
		dataJSON  string
		userIP    string
		userAgent string
		createdAt time.Time
		expiresAt time.Time
	)

	if err := s.db.QueryRow(getSessionByIDQuery, id).Scan(
		&csrfToken,
		&userID,
		&userIP,
		&userAgent,
		&dataJSON,
		&createdAt,
		&expiresAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return session.ErrSessionNotFound
		}

		return fmt.Errorf("scan session %q: %w", id, err)
	}

	var userData map[string]string

	if err := json.Unmarshal([]byte(dataJSON), &userData); err != nil {
		return fmt.Errorf("unmarshal session data %q: %w", id, err)
	}

	if userData == nil {
		userData = make(map[string]string)
	}

	var uid *uuid.UUID

	if userID.Valid {
		parsed, err := uuid.Parse(userID.String)
		if err != nil {
			return fmt.Errorf("parse session user id %q: %w", userID.String, err)
		}

		uid = &parsed
	}

	sess.LoadData(
		id,
		csrfToken,
		uid,
		userIP,
		userAgent,
		userData,
		createdAt,
		expiresAt,
	)

	return nil
}

func (s *Store) DeleteSession(sess *session.Session) error {
	result, err := s.db.Exec(deleteSessionQuery, sess.ID())
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return session.ErrSessionNotFound
	}

	return nil
}
