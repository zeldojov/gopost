package session

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
	"uuid"
)

const createSessionsTableQuery = `
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

const getSessionByIDQuery = `
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

const deleteSessionQuery = `
	DELETE FROM sessions
	WHERE id = ?
`

const saveSessionQuery = `
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

const sessionDuration = 30 * time.Minute
const sessionAbsoluteDuration = 7 * 24 * time.Hour

const csrfFieldName = "csrf_token"

const sessionCookieName = "session_id"
const sessionCookiePath = "/"
const sessionCookieSecure = true
const sessionCookieHTTPOnly = true
const sessionCookieSameSite = http.SameSiteLaxMode

var validRequestMethods = []string{
	http.MethodGet,
	http.MethodPost,
}

var ErrSessionNotFound = errors.New("session not found")
var ErrSessionCookieNotFound = errors.New("session cookie not found")

type contextKey struct{}

type Session struct {
	id        string
	csrfToken string
	userID    *uuid.UUID
	userIP    string
	userAgent string
	userData  map[string]string
	createdAt time.Time
	expiresAt time.Time
}

func NewAnonSession(r *http.Request) *Session {

	s := Session{}

	s.id = newRandomToken()
	s.csrfToken = newRandomToken()

	s.userID = nil
	s.userIP = getClientIP(r)
	s.userAgent = getUserAgent(r)

	s.ClearValues()
	s.Touch()

	return &s
}

func NewAuthSession(userID uuid.UUID, r *http.Request) *Session {

	s := NewAnonSession(r)
	s.userID = &userID

	return s
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

func (s *Session) ClearValues() {
	s.userData = make(map[string]string)
}

func (s *Session) Touch() {
	now := time.Now()

	if s.createdAt.IsZero() {
		s.createdAt = now
	}

	slidingExpiration := now.Add(sessionDuration)
	absoluteExpiration := s.createdAt.Add(sessionAbsoluteDuration)

	if slidingExpiration.After(absoluteExpiration) {
		s.expiresAt = absoluteExpiration
		return
	}

	s.expiresAt = slidingExpiration
}

func (s *Session) IsExpired() bool {
	return time.Now().After(s.expiresAt)
}

func (s *Session) IsAuthenticated() bool {
	return s.userID != nil
}

func (s *Session) IsAnonymous() bool {
	return s.userID == nil
}

func (s *Session) MatchUserAgent(r *http.Request) bool {
	return s.userAgent == getUserAgent(r)
}

func (s *Session) MatchIP(r *http.Request) bool {
	return s.userIP == getClientIP(r)
}

func (s *Session) ShouldRefresh() bool {
	if s.IsExpired() {
		return false
	}

	return time.Until(s.expiresAt) < sessionDuration/2
}

// Save persists the session.
//
// If a session with the same ID does not exist, it is inserted.
// If it already exists, its current values are updated.
func (sess *Session) Save(db *sql.DB) error {
	data, err := json.Marshal(sess.userData)
	if err != nil {
		return err
	}

	_, err = db.Exec(
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

func (sess *Session) Load(db *sql.DB, id string) error {
	var (
		csrfToken string
		userID    sql.NullString
		dataJSON  string
		userIP    string
		userAgent string
		createdAt time.Time
		expiresAt time.Time
	)

	if err := db.QueryRow(getSessionByIDQuery, id).Scan(&csrfToken, &userID, &dataJSON, &userIP, &userAgent, &createdAt, &expiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSessionNotFound
		}

		return fmt.Errorf("scan session %q: %w", id, err)
	}

	var userData map[string]string

	if err := json.Unmarshal([]byte(dataJSON), &userData); err != nil {
		return fmt.Errorf("unmarshal session data %q: %w", id, err)
	}

	if userData == nil {
		userData = make(map[string]string)
	}

	var uid *uuid.UUID

	if userID.Valid {
		parsed, err := uuid.Parse(userID.String)
		if err != nil {
			return fmt.Errorf("parse session user id %q: %w", userID.String, err)
		}

		uid = &parsed
	}

	sess.id = id
	sess.csrfToken = csrfToken
	sess.userID = uid
	sess.userIP = userIP
	sess.userAgent = userAgent
	sess.userData = userData
	sess.createdAt = createdAt
	sess.expiresAt = expiresAt

	return nil
}

func (sess *Session) Destroy(db *sql.DB) error {
	result, err := db.Exec(deleteSessionQuery, sess.id)
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

func (sess *Session) Recreate(db *sql.DB, w http.ResponseWriter, r *http.Request) error {
	if err := sess.Destroy(db); err != nil {
		return fmt.Errorf("destroy session: %w", err)
	}

	unsetSessionCookie(w)

	*sess = *NewAnonSession(r)

	if err := sess.Save(db); err != nil {
		return fmt.Errorf("save recreated session: %w", err)
	}

	setSessionCookie(w, sess.id)

	return nil
}

func CreateSessionsTable(db *sql.DB) error {
	_, err := db.Exec(createSessionsTableQuery)
	return err
}

// GetSession returns the session associated with the request.
// It returns false if no session is stored in the request context.
func GetSession(r *http.Request) (*Session, bool) {
	sess, ok := r.Context().Value(contextKey{}).(*Session)
	return sess, ok
}

func createSession(db *sql.DB, w http.ResponseWriter, r *http.Request) (*Session, error) {
	sess := NewAnonSession(r)

	if err := sess.Save(db); err != nil {
		return nil, fmt.Errorf("save new session: %w", err)
	}

	setSessionCookie(w, sess.id)

	return sess, nil
}

func loadOrCreateSession(db *sql.DB, w http.ResponseWriter, r *http.Request) (*Session, error) {
	cookie, err := getSessionCookie(r)

	if errors.Is(err, ErrSessionCookieNotFound) {
		return createSession(db, w, r)
	}

	if err != nil {
		return nil, fmt.Errorf("get session cookie: %w", err)
	}

	if _, err := uuid.Parse(cookie.Value); err != nil {
		unsetSessionCookie(w)
		return createSession(db, w, r)
	}

	var sess Session

	if err := sess.Load(db, cookie.Value); err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			unsetSessionCookie(w)
			return createSession(db, w, r)
		}

		return nil, fmt.Errorf("load session %q: %w", cookie.Value, err)
	}

	return &sess, nil
}

// Middleware ensures that each request has a valid session and makes it
// available through the request context.
//
// Session changes made by the handler are persisted after the handler
// completes. If logger is non-nil, session errors are logged.
func Middleware(db *sql.DB, logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if !IsValidRequestMethod(r) {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			sess, err := loadOrCreateSession(db, w, r)
			if err != nil {
				if logger != nil {
					logger.Printf("failed to load or create session: %v", err)
				}
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			// If the session has expired, delete it and remove the stale cookie.
			// Then create a new session and set its cookie on the response.
			if sess.IsExpired() {

				if err = sess.Recreate(db, w, r); err != nil {
					if logger != nil {
						logger.Printf("failed to recreate session: %v", err)
					}
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}

			}

			// If the request User-Agent or IP address does not match the session,
			// invalidate the session and remove its cookie. Then create a new session
			// for the current request and set its cookie on the response.
			if !sess.MatchUserAgent(r) || !sess.MatchIP(r) {

				if err = sess.Recreate(db, w, r); err != nil {
					if logger != nil {
						logger.Printf("failed to recreate session: %v", err)
					}
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
			}

			if sess.ShouldRefresh() {
				sess.Touch()
			}

			ctx := context.WithValue(r.Context(), contextKey{}, sess)
			r = r.WithContext(ctx)

			if r.Method == http.MethodPost && !sess.ValidateCSRFToken(r) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)

			if err = sess.Save(db); err != nil {
				if logger != nil {
					logger.Printf("failed to save session %q: %v", sess.id, err)
				}
			}
		})
	}
}

func newRandomToken() string {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return hex.EncodeToString(b)
}

func getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

func getUserAgent(r *http.Request) string {
	return r.UserAgent()
}

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

// setSessionCookie adds a session cookie to the HTTP response.
//
// The cookie is configured using the package session cookie settings
// and contains the provided session ID as its value.
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

// unsetSessionCookie removes the session cookie from the client.
//
// It sends a Set-Cookie header with an expired cookie using the same
// session cookie configuration as the cookie that was originally set.
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

func IsValidRequestMethod(r *http.Request) bool {
	for _, method := range validRequestMethods {
		if r.Method == method {
			return true
		}
	}

	return false
}

func (s *Session) ValidateCSRFToken(r *http.Request) bool {
	if s == nil || s.csrfToken == "" {
		return false
	}

	requestToken := r.PostFormValue(csrfFieldName)

	return requestToken != "" &&
		requestToken == s.csrfToken
}

func GetCSRFToken(r *http.Request) string {
	session, ok := GetSession(r)
	if !ok || session == nil {
		return ""
	}

	if session.csrfToken == "" {
		session.csrfToken = newRandomToken()
	}

	return session.csrfToken
}
