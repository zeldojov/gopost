package session

import (
	"errors"
	"fmt"
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

func LoadSession(r *http.Request) (*Session, error) {

	cookie, err := getSessionCookie(r)
	if err != nil {
		return nil, err
	}

	var sess Session

	if err := sess.Load(cookie.Value); err != nil {
		return nil, err
	}

	return &sess, nil
}

func CreateSession(w http.ResponseWriter, r *http.Request) (*Session, error) {
	sess := NewAnonSession(r)

	if err := sess.Save(); err != nil {
		return nil, fmt.Errorf("save new session: %w", err)
	}

	setSessionCookie(w, sess.id)

	return sess, nil
}

func (sess *Session) Recreate(w http.ResponseWriter, r *http.Request) error {
	userID := sess.userID

	if err := sess.Destroy(); err != nil {
		return err
	}

	UnsetSessionCookie(w)

	*sess = *NewAnonSession(r)
	sess.userID = userID

	if err := sess.Save(); err != nil {
		return err
	}

	setSessionCookie(w, sess.id)

	return nil
}

func SessionDuration() time.Duration {
	return sessionDuration
}

func (sess *Session) ID() string {
	return sess.id
}

func (sess *Session) UserID() *uuid.UUID {
	return sess.UserID()
}
func (sess *Session) ExpiresAt() time.Time {
	return sess.expiresAt
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

func (sess *Session) CSRFToken() string {
	return sess.csrfToken
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
