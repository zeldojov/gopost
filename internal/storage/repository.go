package storage

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("record not found")

type Session struct {
	ID        string
	Data      map[string]string
	IP        string
	UserAgent string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type SessionRepository interface {
	CreateSession(session Session) error
	GetSession(id string) (Session, error)
	UpdateSession(session Session) error
	DeleteSession(id string) error
	RegenerateSession(oldID string, session Session) error
}
