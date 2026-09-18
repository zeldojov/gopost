package store

import (
	"database/sql"
	"errors"
	"fmt"
	"uuid"

	"github.com/zeldojov/gopost/internal/user"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const (
	createUsersTableQuery = `
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);
`
	saveUserQuery = `
INSERT INTO users (
	id,
	username,
	password_hash,
	created_at,
	updated_at
)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	username = excluded.username,
	password_hash = excluded.password_hash,
	updated_at = excluded.updated_at
`
	getUserByUsernameQuery = `
SELECT
	id,
	username,
	password_hash,
	created_at,
	updated_at
FROM users
WHERE username = ?
`
	getUserByIDQuery = `
SELECT
	id,
	username,
	password_hash,
	created_at,
	updated_at
FROM users
WHERE id = ?
`
)

func (s *Store) CreateUsersTable() error {
	_, err := s.db.Exec(createUsersTableQuery)
	return err
}

func (s *Store) SaveUser(u *user.User) error {
	_, err := s.db.Exec(
		saveUserQuery,
		u.ID.String(),
		u.Username,
		u.PasswordHash,
		u.CreatedAt,
		u.UpdatedAt,
	)

	if err == nil {
		return nil
	}

	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return err
	}

	if sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
		return user.ErrUsernameTaken
	}

	return err
}

func (s *Store) GetUserByID(id uuid.UUID) (*user.User, error) {
	var u user.User

	err := s.db.QueryRow(
		getUserByIDQuery,
		id.String(),
	).Scan(
		&u.ID,
		&u.Username,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}

		return nil, fmt.Errorf("get user by id %q: %w", id, err)
	}

	return &u, nil
}

func (s *Store) GetUserByUsername(username string) (*user.User, error) {
	var u user.User

	err := s.db.QueryRow(
		getUserByUsernameQuery,
		username,
	).Scan(
		&u.ID,
		&u.Username,
		&u.PasswordHash,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}

		return nil, fmt.Errorf("get user by username %q: %w", username, err)
	}

	return &u, nil
}
