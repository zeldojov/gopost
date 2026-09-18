package session

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"uuid"
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

var DB *sql.DB

func CreateSessionsTable(db *sql.DB) error {
	if _, err := db.Exec(createSessionsTableQuery); err != nil {
		db.Close()
		return err
	}
	return nil
}

func (sess *Session) Save() error {
	data, err := json.Marshal(sess.userData)
	if err != nil {
		return err
	}

	_, err = DB.Exec(
		saveSessionQuery,
		sess.id,
		sess.csrfToken,
		sess.userID,
		sess.userIP,
		sess.userAgent,
		data,
		sess.createdAt,
		sess.expiresAt,
	)

	return err
}

func (sess *Session) Load(id string) error {

	var (
		csrfToken string
		userID    sql.NullString
		dataJSON  string
		userIP    string
		userAgent string
		createdAt time.Time
		expiresAt time.Time
	)

	if err := DB.QueryRow(getSessionByIDQuery, id).Scan(
		&csrfToken,
		&userID,
		&userIP,
		&userAgent,
		&dataJSON,
		&createdAt,
		&expiresAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSessionNotFound
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

	newSess := &Session{
		id:        id,
		csrfToken: csrfToken,
		userID:    uid,
		userIP:    userIP,
		userAgent: userAgent,
		userData:  userData,
		createdAt: createdAt,
		expiresAt: expiresAt,
	}

	*sess = *newSess

	return nil
}

func (sess *Session) Destroy() error {
	result, err := DB.Exec(deleteSessionQuery, sess.id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrSessionNotFound
	}

	return nil
}
