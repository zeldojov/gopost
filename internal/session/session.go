package session

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"time"
)

// region setup

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
		UserAgent: r.UserAgent(),
		CreatedAt: now,
		ExpiresAt: now.Add(sessionDuration),
	}
}

// endregion setup
// region session

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

// endregion session

// region helpers

func GetSession(r *http.Request) (*Session, bool) {
	sess, ok := r.Context().Value(contextKey{}).(*Session)
	return sess, ok
}

func newRandomToken() string {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return hex.EncodeToString(b)
}

func newSessionID() string {
	return newRandomToken()
}

func newCSRFToken() string {
	return newRandomToken()
}

func getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

func GetSessionID(r *http.Request, config CookieConfig) (string, error) {
	cookie, err := r.Cookie(config.Name)
	if errors.Is(err, http.ErrNoCookie) {
		return "", ErrSessionCookieNotFound
	}

	if err != nil {
		return "", err
	}

	return cookie.Value, nil
}

func SetSessionCookie(w http.ResponseWriter, config CookieConfig, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     config.Name,
		Value:    sessionID,
		Path:     config.Path,
		HttpOnly: config.HTTPOnly,
		Secure:   config.Secure,
		SameSite: config.SameSite,
	})
}

func DeleteSessionCookie(w http.ResponseWriter, config CookieConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     config.Name,
		Value:    "",
		Path:     config.Path,
		MaxAge:   -1,
		HttpOnly: config.HTTPOnly,
		Secure:   config.Secure,
		SameSite: config.SameSite,
	})
}

func (s *Session) GetCSRFToken() string {
	if token, ok := s.Get("csrf_token"); ok {
		return token
	}

	token := newCSRFToken()
	s.Set("csrf_token", token)

	return token
}

// endregion helpers
