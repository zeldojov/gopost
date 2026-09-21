package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/go-sql-driver/mysql"
	"github.com/zeldojov/gopost/internal/user"
)

const (
	createUsersTableQuery = `
CREATE TABLE IF NOT EXISTS users (
	id CHAR(36) NOT NULL,
	jmbg CHAR(13) NOT NULL,
	username VARCHAR(16) NOT NULL,
	full_name VARCHAR(64) NOT NULL,
	email VARCHAR(254) NOT NULL,
	password_hash VARCHAR(255) NOT NULL,
	active BOOLEAN NOT NULL DEFAULT TRUE,
	email_verified_at DATETIME NULL,
	password_changed_at DATETIME NOT NULL,
	last_login_at DATETIME NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,

	PRIMARY KEY (id),
	UNIQUE KEY uq_users_jmbg (jmbg),
	UNIQUE KEY uq_users_username (username),
	UNIQUE KEY uq_users_email (email)
);
`

	createUserQuery = `
INSERT INTO users (
	id,
	jmbg,
	username,
	full_name,
	email,
	password_hash,
	active,
	email_verified_at,
	password_changed_at,
	last_login_at,
	created_at,
	updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	updateUserQuery = `
UPDATE users
SET
	full_name = ?,
	updated_at = ?
WHERE id = ?
`

	updateUsernameQuery = `
UPDATE users
SET
	username = ?,
	updated_at = ?
WHERE id = ?
`

	updateEmailQuery = `
UPDATE users
SET
	email = ?,
	updated_at = ?
WHERE id = ?
`

	updatePasswordQuery = `
UPDATE users
SET
	password_hash = ?,
	updated_at = ?
WHERE id = ?
`

	getUserByUsernameQuery = `
SELECT
	id,
	jmbg,
	username,
	full_name,
	email,
	password_hash,
	active,
	email_verified_at,
	password_changed_at,
	last_login_at,
	created_at,
	updated_at
FROM users
WHERE username = ?
`

	getUserByIDQuery = `
SELECT
	id,
	jmbg,
	username,
	full_name,
	email,
	password_hash,
	active,
	email_verified_at,
	password_changed_at,
	last_login_at,
	created_at,
	updated_at
FROM users
WHERE id = ?
`

	deleteUserQuery = `
DELETE FROM users
WHERE id = ?
`

	updateLastLoginQuery = `
UPDATE users
SET
	last_login_at = ?,
	updated_at = ?
WHERE id = ?
`
)

func (s *Store) CreateUser(u *user.User) error {
	_, err := s.db.Exec(
		createUserQuery,
		u.ID().String(),
		u.JMBG(),
		u.Username(),
		u.FullName(),
		u.Email(),
		u.PasswordHash(),
		u.Active(),
		u.EmailVerifiedAt(),
		u.PasswordChangedAt(),
		u.LastLoginAt(),
		u.CreatedAt(),
		u.UpdatedAt(),
	)
	if err == nil {
		return nil
	}

	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) {
		return fmt.Errorf("create user: %w", err)
	}

	if mysqlErr.Number == 1062 {
		switch {
		case strings.Contains(mysqlErr.Message, "uq_users_jmbg"):
			return user.ErrJMBGTaken
		case strings.Contains(mysqlErr.Message, "uq_users_username"):
			return user.ErrUsernameTaken
		case strings.Contains(mysqlErr.Message, "uq_users_email"):
			return user.ErrEmailTaken
		}
	}

	return fmt.Errorf("create user: %w", err)
}

func (s *Store) UpdateUser(u *user.User) error {
	_, err := s.db.Exec(
		updateUserQuery,
		u.FullName(),
		u.UpdatedAt(),
		u.ID().String(),
	)
	if err != nil {
		return fmt.Errorf("update user %q: %w", u.ID(), err)
	}

	return nil
}

func (s *Store) UpdateUsername(u *user.User) error {
	_, err := s.db.Exec(
		updateUsernameQuery,
		u.Username(),
		u.UpdatedAt(),
		u.ID().String(),
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError

		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return user.ErrUsernameTaken
		}

		return fmt.Errorf("update username for user %q: %w", u.ID(), err)
	}

	return nil
}

func (s *Store) UpdateEmail(u *user.User) error {
	_, err := s.db.Exec(
		updateEmailQuery,
		u.Email(),
		u.UpdatedAt(),
		u.ID().String(),
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError

		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return user.ErrEmailTaken
		}

		return fmt.Errorf("update email for user %q: %w", u.ID(), err)
	}

	return nil
}

func (s *Store) UpdatePassword(u *user.User) error {
	_, err := s.db.Exec(
		updatePasswordQuery,
		u.PasswordHash(),
		u.UpdatedAt(),
		u.ID().String(),
	)
	if err != nil {
		return fmt.Errorf("update password for user %q: %w", u.ID(), err)
	}

	return nil
}

func (s *Store) GetUserByID(id uuid.UUID) (*user.User, error) {
	var (
		userID            uuid.UUID
		jmbg              string
		username          string
		fullName          string
		email             string
		passwordHash      string
		active            bool
		emailVerifiedAt   *time.Time
		passwordChangedAt time.Time
		lastLoginAt       *time.Time
		createdAt         time.Time
		updatedAt         time.Time
	)

	err := s.db.QueryRow(
		getUserByIDQuery,
		id.String(),
	).Scan(
		&userID,
		&jmbg,
		&username,
		&fullName,
		&email,
		&passwordHash,
		&active,
		&emailVerifiedAt,
		&passwordChangedAt,
		&lastLoginAt,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}

		return nil, fmt.Errorf("get user by id %q: %w", id, err)
	}

	return user.LoadUser(
		userID,
		jmbg,
		username,
		fullName,
		email,
		passwordHash,
		active,
		emailVerifiedAt,
		passwordChangedAt,
		lastLoginAt,
		createdAt,
		updatedAt,
	), nil
}

func (s *Store) GetUserByUsername(username string) (*user.User, error) {
	var (
		userID            uuid.UUID
		userUsername      string
		jmbg              string
		fullName          string
		email             string
		passwordHash      string
		active            bool
		emailVerifiedAt   *time.Time
		passwordChangedAt time.Time
		lastLoginAt       *time.Time
		createdAt         time.Time
		updatedAt         time.Time
	)

	err := s.db.QueryRow(
		getUserByUsernameQuery,
		username,
	).Scan(
		&userID,
		&jmbg,
		&userUsername,
		&fullName,
		&email,
		&passwordHash,
		&active,
		&emailVerifiedAt,
		&passwordChangedAt,
		&lastLoginAt,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, user.ErrUserNotFound
		}

		return nil, fmt.Errorf("get user by username %q: %w", username, err)
	}

	return user.LoadUser(
		userID,
		jmbg,
		userUsername,
		fullName,
		email,
		passwordHash,
		active,
		emailVerifiedAt,
		passwordChangedAt,
		lastLoginAt,
		createdAt,
		updatedAt,
	), nil
}

func (s *Store) DeleteUser(id uuid.UUID) error {
	result, err := s.db.Exec(
		deleteUserQuery,
		id.String(),
	)
	if err != nil {
		return fmt.Errorf("delete user %q: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete user %q: %w", id, err)
	}

	if rows == 0 {
		return user.ErrUserNotFound
	}

	return nil
}

func (s *Store) UpdateLastLogin(u *user.User) error {
	_, err := s.db.Exec(
		updateLastLoginQuery,
		u.LastLoginAt(),
		u.UpdatedAt(),
		u.ID().String(),
	)
	if err != nil {
		return fmt.Errorf("update last login for user %q: %w", u.ID(), err)
	}

	return nil
}
