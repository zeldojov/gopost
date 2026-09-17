package main

import (
	"context"
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
	sessionCookieSecure   = true
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

	newSess, err := dbLoadSession(id)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return ErrSessionNotFound
		}

		return err
	}

	*sess = *newSess

	return nil
}

func (sess *Session) Destroy() error {
	return dbDeleteSession(sess)
}

func (sess *Session) Recreate(w http.ResponseWriter, r *http.Request) error {
	if err := sess.Destroy(); err != nil {
		return err
	}

	unsetSessionCookie(w)

	*sess = *NewAnonSession(r)

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
	cookie, err := getSessionCookie(r)
	if err != nil {
		return nil, err
	}

	if _, err := uuid.Parse(cookie.Value); err != nil {
		return nil, ErrSessionNotFound
	}

	var sess Session

	if err := sess.Load(cookie.Value); err != nil {
		return nil, err
	}

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

func BaseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sess *Session
		var err error

		// 1 Load session
		switch r.Method {
		case http.MethodGet:
			sess, err = loadSession(r)

			if errors.Is(err, ErrSessionCookieNotFound) {
				sess, err = createSession(w, r)
			} else if errors.Is(err, ErrSessionNotFound) {
				unsetSessionCookie(w)
				sess, err = createSession(w, r)
			}

		case http.MethodPost:
			sess, err = loadSession(r)

			if errors.Is(err, ErrSessionCookieNotFound) || errors.Is(err, ErrSessionNotFound) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err != nil {

			LOG.Printf("failed to load session: %v", err)

			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// 2 Validate session / Recover session
		switch r.Method {
		case http.MethodGet:
			if sess.IsExpired() || !sess.MatchUserAgent(r) || !sess.MatchIP(r) {
				if err = sess.Recreate(w, r); err != nil {

					LOG.Printf("failed to recreate session: %v", err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
			}

		case http.MethodPost:
			if sess.IsExpired() || !sess.MatchUserAgent(r) || !sess.MatchIP(r) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}

		// 3 CSRF
		if r.Method == http.MethodPost && !sess.ValidateCSRFToken(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		// 4 Refresh
		if sess.ShouldRefresh() {
			sess.Touch()
		}

		// 5 Context
		ctx := context.WithValue(r.Context(), contextKey{}, sess)
		r = r.WithContext(ctx)

		// 6 Handler
		next.ServeHTTP(w, r)

		// 7 Save session
		if err = sess.Save(); err != nil {
			LOG.Printf("failed to save session %q: %v", sess.id, err)
		}
	})
}

// endregion middleware
