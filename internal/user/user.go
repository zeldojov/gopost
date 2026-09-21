package user

import (
	"errors"
	"time"
	"uuid"
)

var (
	ErrUsernameTaken = errors.New("username already exists")
	ErrEmailTaken    = errors.New("email already exists")
	ErrJMBGTaken     = errors.New("jmbg already exists")
	ErrUserNotFound  = errors.New("user not found")
)

type User struct {
	id                uuid.UUID
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
}

func CreateUser(
	jmbg string,
	username string,
	fullName string,
	email string,
	passwordHash string,
) *User {
	now := time.Now()

	return &User{
		id:                uuid.New(),
		jmbg:              jmbg,
		username:          username,
		fullName:          fullName,
		email:             email,
		passwordHash:      passwordHash,
		active:            true,
		emailVerifiedAt:   nil,
		passwordChangedAt: now,
		lastLoginAt:       nil,
		createdAt:         now,
		updatedAt:         now,
	}
}

func LoadUser(
	id uuid.UUID,
	jmbg string,
	username string,
	fullName string,
	email string,
	passwordHash string,
	active bool,
	emailVerifiedAt *time.Time,
	passwordChangedAt time.Time,
	lastLoginAt *time.Time,
	createdAt time.Time,
	updatedAt time.Time,
) *User {
	return &User{
		id:                id,
		jmbg:              jmbg,
		username:          username,
		fullName:          fullName,
		email:             email,
		passwordHash:      passwordHash,
		active:            active,
		emailVerifiedAt:   emailVerifiedAt,
		passwordChangedAt: passwordChangedAt,
		lastLoginAt:       lastLoginAt,
		createdAt:         createdAt,
		updatedAt:         updatedAt,
	}
}

func (u *User) RecordLogin() {
	now := time.Now()

	u.lastLoginAt = &now
	u.updatedAt = now
}

func (u *User) VerifyEmail() {
	if u.emailVerifiedAt != nil {
		return
	}

	now := time.Now()

	u.emailVerifiedAt = &now
	u.updatedAt = now
}

func (u *User) Activate() {
	if u.active {
		return
	}

	u.active = true
	u.updatedAt = time.Now()
}

func (u *User) Deactivate() {
	if !u.active {
		return
	}

	u.active = false
	u.updatedAt = time.Now()
}

func (u *User) ChangeFullName(fullName string) {
	u.fullName = fullName
	u.updatedAt = time.Now()
}

func (u *User) ChangeUsername(username string) {
	u.username = username
	u.updatedAt = time.Now()
}

func (u *User) ChangeEmail(email string) {
	u.email = email
	u.emailVerifiedAt = nil
	u.updatedAt = time.Now()
}

func (u *User) ChangePassword(passwordHash string) {
	now := time.Now()

	u.passwordHash = passwordHash
	u.passwordChangedAt = now
	u.updatedAt = now
}

func (u *User) ID() uuid.UUID {
	return u.id
}

func (u *User) JMBG() string {
	return u.jmbg
}

func (u *User) Username() string {
	return u.username
}

func (u *User) FullName() string {
	return u.fullName
}

func (u *User) Email() string {
	return u.email
}

func (u *User) PasswordHash() string {
	return u.passwordHash
}

func (u *User) Active() bool {
	return u.active
}

func (u *User) EmailVerifiedAt() *time.Time {
	return u.emailVerifiedAt
}

func (u *User) PasswordChangedAt() time.Time {
	return u.passwordChangedAt
}

func (u *User) LastLoginAt() *time.Time {
	return u.lastLoginAt
}

func (u *User) CreatedAt() time.Time {
	return u.createdAt
}

func (u *User) UpdatedAt() time.Time {
	return u.updatedAt
}
