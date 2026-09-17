package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
	"uuid"
)

// region definition
const (
	createSessionsTableQuery = `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		csrf_token TEXT NOT NULL,
		user_id TEXT NULL,
		user_ip TEXT NOT NULL,
		user_agent TEXT NOT NULL,
		user_data TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		expires_at DATETIME NOT NULL
	);
`
	saveSessionQuery = `
	INSERT INTO sessions (
		id,
		csrf_token,
		user_id,
		user_ip,
		user_agent,
		user_data,
		created_at,
		expires_at
	)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		csrf_token = excluded.csrf_token,
		user_id = excluded.user_id,
		user_ip = excluded.user_ip,
		user_agent = excluded.user_agent,
		user_data = excluded.user_data,
		expires_at = excluded.expires_at
`
	getSessionByIDQuery = `
	SELECT
		csrf_token,
		user_id,
		user_ip,
		user_agent,
		user_data,
		created_at,
		expires_at
	FROM sessions
	WHERE id = ?
`
	deleteSessionQuery = `
	DELETE FROM sessions
	WHERE id = ?
`
	sessionDuration         = 30 * time.Minute
	sessionAbsoluteDuration = 7 * 24 * time.Hour

	csrfFieldName = "csrf_token"

	sessionCookieName     = "session_id"
	sessionCookiePath     = "/"
	sessionCookieSecure   = false
	sessionCookieHTTPOnly = true
	sessionCookieSameSite = http.SameSiteLaxMode
)

var (
	ErrSessionCookieNotFound = errors.New("session cookie not found")
	ErrSessionNotFound       = errors.New("session not found")
)

type (
	contextKey struct{}

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

// endregion definition
// region construct
func NewAnonSession(r *http.Request) *Session {

	s := Session{
		id:        NewRandomToken(),
		csrfToken: NewRandomToken(),
		userID:    nil,
		userIP:    GetClientIP(r),
		userAgent: GetUserAgent(r),
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

// endregion construct
// region db helpers
func dbSaveSession(sess *Session, data []byte) error {
	_, err := DB.Exec(
		saveSessionQuery,
		sess.id,
		sess.csrfToken,
		sess.userID,
		sess.userIP,
		sess.userAgent,
		data,
		sess.createdAt,
		sess.expiresAt,
	)

	return err
}

func dbLoadSession(id string) (*Session, error) {

	var (
		csrfToken string
		userID    sql.NullString
		dataJSON  string
		userIP    string
		userAgent string
		createdAt time.Time
		expiresAt time.Time
	)
	LOG.Printf(
		"DB LOAD 1: requested_id=%q",
		id,
	)
	if err := DB.QueryRow(getSessionByIDQuery, id).Scan(
		&csrfToken,
		&userID,
		&userIP,
		&userAgent,
		&dataJSON,
		&createdAt,
		&expiresAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}

		return nil, fmt.Errorf("scan session %q: %w", id, err)
	}

	var userData map[string]string

	if err := json.Unmarshal([]byte(dataJSON), &userData); err != nil {
		return nil, fmt.Errorf("unmarshal session data %q: %w", id, err)
	}

	if userData == nil {
		userData = make(map[string]string)
	}

	var uid *uuid.UUID

	if userID.Valid {
		parsed, err := uuid.Parse(userID.String)
		if err != nil {
			return nil, fmt.Errorf("parse session user id %q: %w", userID.String, err)
		}

		uid = &parsed
	}

	sess := &Session{
		id:        id,
		csrfToken: csrfToken,
		userID:    uid,
		userIP:    userIP,
		userAgent: userAgent,
		userData:  userData,
		createdAt: createdAt,
		expiresAt: expiresAt,
	}
	LOG.Printf(
		"DB LOAD 2: requested_id=%q sess.id=%q sess=%p",
		id,
		sess.id,
		sess,
	)
	return sess, nil
}

func dbDeleteSession(sess *Session) error {
	result, err := DB.Exec(deleteSessionQuery, sess.id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// endregion db helpers
// region database
func (sess *Session) Save() error {
	data, err := json.Marshal(sess.userData)
	if err != nil {
		return err
	}

	return dbSaveSession(sess, data)
}

func (sess *Session) Load(id string) error {
	LOG.Printf(
		"SESSION LOAD 1: receiver=%p requested_id=%q current_id=%q",
		sess,
		id,
		sess.id,
	)

	newSess, err := dbLoadSession(id)
	if err != nil {
		return err
	}

	LOG.Printf(
		"SESSION LOAD 2: newSess=%p newSess.id=%q",
		newSess,
		newSess.id,
	)

	*sess = *newSess

	LOG.Printf(
		"SESSION LOAD 3: receiver=%p sess.id=%q",
		sess,
		sess.id,
	)

	return nil
}

func (sess *Session) Destroy() error {
	return dbDeleteSession(sess)
}

func (sess *Session) Recreate(w http.ResponseWriter, r *http.Request) error {
	userID := sess.userID

	if err := sess.Destroy(); err != nil {
		return err
	}

	unsetSessionCookie(w)

	*sess = *NewAnonSession(r)
	sess.userID = userID

	if err := sess.Save(); err != nil {
		return err
	}

	setSessionCookie(w, sess.id)

	return nil
}

// endregion database
// region cookies
func getSessionCookie(r *http.Request) (*http.Cookie, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if errors.Is(err, http.ErrNoCookie) {
		return nil, ErrSessionCookieNotFound
	}
	if err != nil {
		return nil, err
	}

	return cookie, nil
}

func setSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     sessionCookiePath,
		HttpOnly: sessionCookieHTTPOnly,
		Secure:   sessionCookieSecure,
		SameSite: sessionCookieSameSite,
	})
}

func unsetSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     sessionCookiePath,
		MaxAge:   -1,
		HttpOnly: sessionCookieHTTPOnly,
		Secure:   sessionCookieSecure,
		SameSite: sessionCookieSameSite,
	})
}

// endregion cookies
// region methods
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
	return s.userAgent == GetUserAgent(r)
}

func (s *Session) MatchIP(r *http.Request) bool {
	return s.userIP == GetClientIP(r)
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

func (s *Session) ValidateCSRFToken(r *http.Request) bool {
	if s.csrfToken == "" {
		return false
	}

	requestToken := r.PostFormValue(csrfFieldName)

	return requestToken != "" &&
		requestToken == s.csrfToken
}

func (s *Session) GetValue(key string) (string, bool) {
	value, ok := s.userData[key]
	return value, ok
}

func (s *Session) SetValue(key, value string) {
	s.userData[key] = value
}

func (s *Session) DeleteValue(key string) {
	delete(s.userData, key)
}

func (s *Session) HasValue(key string) bool {
	_, ok := s.userData[key]
	return ok
}

func (s *Session) IsAuthenticated() bool {
	return s.userID != nil
}

func (s *Session) IsAnonymous() bool {
	return s.userID == nil
}

// endregion methods
// region middleware

func loadSession(r *http.Request) (*Session, error) {
	LOG.Printf("========== loadSession NEW CODE ==========")

	cookie, err := getSessionCookie(r)
	if err != nil {
		LOG.Printf("COOKIE READ ERROR: %v", err)
		return nil, err
	}

	LOG.Printf("LOAD 1: cookie=%q", cookie.Value)

	var sess Session

	LOG.Printf("LOAD 2: before Load sess.id=%q", sess.id)

	if err := sess.Load(cookie.Value); err != nil {
		return nil, err
	}

	LOG.Printf(
		"LOAD 3: cookie=%q sess.id=%q sess=%p",
		cookie.Value,
		sess.id,
		&sess,
	)

	return &sess, nil
}

func createSession(w http.ResponseWriter, r *http.Request) (*Session, error) {
	sess := NewAnonSession(r)

	if err := sess.Save(); err != nil {
		return nil, fmt.Errorf("save new session: %w", err)
	}

	setSessionCookie(w, sess.id)

	return sess, nil
}

// endregion middleware
// region auth
func (sess *Session) Authenticate(userID uuid.UUID, w http.ResponseWriter, r *http.Request) error {
	if err := sess.Destroy(); err != nil {
		return err
	}

	unsetSessionCookie(w)

	*sess = *NewAuthSession(userID, r)

	if err := sess.Save(); err != nil {
		return err
	}

	setSessionCookie(w, sess.id)

	return nil
}

// endregion auth
