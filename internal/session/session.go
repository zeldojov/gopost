package session

import (
	"errors"
	"net/http"
	"time"
	"uuid"

	"github.com/zeldojov/gopost/internal/utils"
)

const (
	sessionDuration         = 30 * time.Minute
	sessionAbsoluteDuration = 7 * 24 * time.Hour
)

type (
	ContextKey struct{}

	Session struct {
		id        string
		csrfToken string
		userID    *uuid.UUID
		userIP    string
		userAgent string
		userData  map[string]string
		createdAt time.Time
		expiresAt time.Time
	}
)

var (
	ErrSessionCookieNotFound = errors.New("session cookie not found")
	ErrSessionNotFound       = errors.New("session not found")
)

func NewAnonSession(r *http.Request) *Session {
	s := Session{
		id:        utils.NewRandomToken(),
		csrfToken: utils.NewRandomToken(),
		userID:    nil,
		userIP:    utils.GetClientIP(r),
		userAgent: utils.GetUserAgent(r),
	}

	s.ClearValues()
	s.Touch()

	return &s
}

func NewAuthSession(userID uuid.UUID, r *http.Request) *Session {
	s := NewAnonSession(r)
	s.userID = &userID

	return s
}

func SessionDuration() time.Duration {
	return sessionDuration
}

func (sess *Session) Recreate(w http.ResponseWriter, r *http.Request) error {
	userID := sess.userID

	// DB deletion is now done by store.
	// This method only changes the session state.
	UnsetSessionCookie(w)

	*sess = *NewAnonSession(r)
	sess.userID = userID

	SetSessionCookie(w, sess.id)

	return nil
}

func (sess *Session) SetExpiresAt(when time.Time) {
	sess.expiresAt = when
}

func (sess *Session) SetUserIP(ip string) {
	sess.userIP = ip
}

func (sess *Session) SetUserAgent(agent string) {
	sess.userAgent = agent
}

func (sess *Session) ClearValues() {
	sess.userData = make(map[string]string)
}

func (sess *Session) Touch() {
	now := time.Now()

	if sess.createdAt.IsZero() {
		sess.createdAt = now
	}

	slidingExpiration := now.Add(sessionDuration)
	absoluteExpiration := sess.createdAt.Add(sessionAbsoluteDuration)

	if slidingExpiration.After(absoluteExpiration) {
		sess.expiresAt = absoluteExpiration
	} else {
		sess.expiresAt = slidingExpiration
	}
}

func (s *Session) MatchUserAgent(r *http.Request) bool {
	return s.userAgent == utils.GetUserAgent(r)
}

func (s *Session) MatchIP(r *http.Request) bool {
	return s.userIP == utils.GetClientIP(r)
}

func (s *Session) IsExpired() bool {
	return time.Now().After(s.expiresAt)
}

func (s *Session) ShouldRefresh() bool {
	if s.IsExpired() {
		return false
	}

	return time.Until(s.expiresAt) < sessionDuration/2
}

func (s *Session) ID() string {
	return s.id
}

func (s *Session) CSRFToken() string {
	return s.csrfToken
}

func (s *Session) UserID() *uuid.UUID {
	return s.userID
}

func (s *Session) UserIP() string {
	return s.userIP
}

func (s *Session) UserAgent() string {
	return s.userAgent
}

func (s *Session) UserData() map[string]string {
	return s.userData
}

func (s *Session) CreatedAt() time.Time {
	return s.createdAt
}

func (s *Session) ExpiresAt() time.Time {
	return s.expiresAt
}
func (sess *Session) LoadData(
	id string,
	csrfToken string,
	userID *uuid.UUID,
	userIP string,
	userAgent string,
	userData map[string]string,
	createdAt time.Time,
	expiresAt time.Time,
) {
	sess.id = id
	sess.csrfToken = csrfToken
	sess.userID = userID
	sess.userIP = userIP
	sess.userAgent = userAgent
	sess.userData = userData
	sess.createdAt = createdAt
	sess.expiresAt = expiresAt
}
