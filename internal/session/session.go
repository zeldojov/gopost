package session

import (
	"errors"
	"net/http"
	"time"
)

const sessionDuration = 30 * time.Minute
const sessionAbsoluteDuration = 7 * 24 * time.Hour

var ErrSessionCookieNotFound = errors.New("session cookie not found")
var ErrSessionNotFound = errors.New("session not found")
var ErrSessionExpired = errors.New("session expired")

type contextKey struct{}
type CookieConfig struct {
	Name     string
	Path     string
	Secure   bool
	HTTPOnly bool
	SameSite http.SameSite
}

type Session struct {
	Data      map[string]string
	IP        string
	UserAgent string
	CreatedAt time.Time
	ExpiresAt time.Time
}

func newSession(r *http.Request) Session {
	now := time.Now()
	return Session{
		Data:      make(map[string]string),
		IP:        getClientIP(r),
		UserAgent: getUserAgent(r),
		CreatedAt: now,
		ExpiresAt: now.Add(sessionDuration),
	}
}

func (s *Session) Get(key string) (string, bool) {
	value, ok := s.Data[key]
	return value, ok
}

func (s *Session) Set(key, value string) {
	s.Data[key] = value
}

func (s *Session) Delete(key string) {
	delete(s.Data, key)
}

func (s *Session) Has(key string) bool {
	_, ok := s.Data[key]
	return ok
}

// region helpers

func GetSession(r *http.Request) (*Session, bool) {
	sess, ok := r.Context().Value(contextKey{}).(*Session)
	return sess, ok
}

func newSessionID() string {
	return newRandomToken()
}

// endregion helpers
