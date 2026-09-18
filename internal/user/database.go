package user

import (
	"database/sql"
	"errors"
	"fmt"
	"uuid"

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

var (
	ErrUsernameTaken = errors.New("username already exists")
	ErrUserNotFound  = errors.New("user not found")
)

var DB *sql.DB

func CreateUsersTable(sqlite *sql.DB) error {
	if _, err := sqlite.Exec(createUsersTableQuery); err != nil {
		return err
	}

	return nil
}

func (u *User) Save() error {
	_, err := DB.Exec(
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
		return ErrUsernameTaken
	}

	return err
}

func GetUserByID(id uuid.UUID) (*User, error) {
	var user User

	err := DB.QueryRow(
		getUserByIDQuery,
		id.String(),
	).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("get user by id %q: %w", id, err)
	}

	return &user, nil
}

func GetUserByUsername(username string) (*User, error) {
	var user User

	err := DB.QueryRow(
		getUserByUsernameQuery,
		username,
	).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("get user by username %q: %w", username, err)
	}

	return &user, nil
}
