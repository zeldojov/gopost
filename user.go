package main

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	"golang.org/x/crypto/argon2"
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

	argon2Memory      = 64 * 1024 // 64 MiB
	argon2Iterations  = 3
	argon2Parallelism = 2
	argon2SaltLength  = 16
	argon2KeyLength   = 32
)

var (
	errPasswordTooShort       = errors.New("password must be at least 12 characters")
	errPasswordTooLong        = errors.New("password must be at most 128 characters")
	errPasswordInvalidChars   = errors.New("password contains invalid characters")
	errPasswordMissingUpper   = errors.New("password must contain an uppercase letter")
	errPasswordMissingLower   = errors.New("password must contain a lowercase letter")
	errPasswordMissingDigit   = errors.New("password must contain a digit")
	errPasswordMissingSpecial = errors.New("password must contain a special character")

	errUsernameTooShort          = errors.New("username must be at least 3 characters")
	errUsernameTooLong           = errors.New("username must be at most 32 characters")
	errUsernameInvalidChars      = errors.New("username contains invalid characters")
	errUsernameInvalidUnderscore = errors.New("username contains invalid underscore placement")
	errUsernameTaken             = errors.New("username already exists")
	errUserNotFound              = errors.New("user not found")
)

type User struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func createUsersTable(sqlite *sql.DB) error {
	if _, err := sqlite.Exec(createUsersTableQuery); err != nil {
		return err
	}

	return nil
}

func validatePassword(password string) error {
	if len(password) < 12 {
		return errPasswordTooShort
	}

	if len(password) > 128 {
		return errPasswordTooLong
	}

	var upper, lower, digit, special bool

	for i := 0; i < len(password); i++ {
		c := password[i]

		switch {
		case c >= 65 && c <= 90:
			upper = true

		case c >= 97 && c <= 122:
			lower = true

		case c >= 48 && c <= 57:
			digit = true

		case c >= 33 && c <= 47 ||
			c >= 58 && c <= 64 ||
			c >= 91 && c <= 96 ||
			c >= 123 && c <= 126:
			special = true

		default:
			return errPasswordInvalidChars
		}
	}

	if !upper {
		return errPasswordMissingUpper
	}

	if !lower {
		return errPasswordMissingLower
	}

	if !digit {
		return errPasswordMissingDigit
	}

	if !special {
		return errPasswordMissingSpecial
	}

	return nil
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLength)

	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argon2Iterations,
		argon2Memory,
		argon2Parallelism,
		argon2KeyLength,
	)

	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory,
		argon2Iterations,
		argon2Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func verifyPassword(password, encodedHash string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false
	}

	if parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}

	var memory, iterations uint32
	var parallelism uint8

	if _, err := fmt.Sscanf(
		parts[3],
		"m=%d,t=%d,p=%d",
		&memory,
		&iterations,
		&parallelism,
	); err != nil {
		return false
	}

	if memory != argon2Memory ||
		iterations != argon2Iterations ||
		parallelism != argon2Parallelism {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	if len(salt) != argon2SaltLength {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	if len(expectedHash) != argon2KeyLength {
		return false
	}

	actualHash := argon2.IDKey(
		[]byte(password),
		salt,
		iterations,
		memory,
		parallelism,
		argon2KeyLength,
	)

	return subtle.ConstantTimeCompare(actualHash, expectedHash) == 1
}

func validateUsername(username string) error {
	if len(username) < 3 {
		return errUsernameTooShort
	}

	if len(username) > 32 {
		return errUsernameTooLong
	}

	for i := 0; i < len(username); i++ {
		c := username[i]

		switch {
		case c >= 65 && c <= 90:
		case c >= 97 && c <= 122:
		case c >= 48 && c <= 57:

		case c == 95:
			if i == 0 || i == len(username)-1 {
				return errUsernameInvalidUnderscore
			}

			if username[i-1] == 95 {
				return errUsernameInvalidUnderscore
			}

		default:
			return errUsernameInvalidChars
		}
	}

	return nil
}

func CreateUser(username, passwordHash string) *User {
	now := time.Now()

	return &User{
		ID:           uuid.New(),
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
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
		return errUsernameTaken
	}

	return err
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
			return nil, errUserNotFound
		}

		return nil, fmt.Errorf("get user by username %q: %w", username, err)
	}

	return &user, nil
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
			return nil, errUserNotFound
		}

		return nil, fmt.Errorf("get user by id %q: %w", id, err)
	}

	return &user, nil
}
