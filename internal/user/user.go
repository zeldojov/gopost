package user

import (
	"time"
	"uuid"
)

type User struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
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
