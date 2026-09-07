package session

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"time"

	_ "modernc.org/sqlite"
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

type Store struct {
	db *sql.DB
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

func NewStore(db *sql.DB) *Store {
	return &Store{
		db: db,
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

// region store

func (s *Store) AddSession(r *http.Request) (string, error) {
	sessionID := newSessionID()
	session := newSession(r)

	data, err := json.Marshal(session.Data)
	if err != nil {
		return "", err
	}

	_, err = s.db.Exec(`
		INSERT INTO sessions (
			id,
			data,
			ip,
			user_agent,
			created_at,
			expires_at
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		sessionID,
		data,
		session.IP,
		session.UserAgent,
		session.CreatedAt,
		session.ExpiresAt,
	)
	if err != nil {
		return "", err
	}

	return sessionID, nil
}
func (s *Store) RemoveSession(sessionID string) error {
	result, err := s.db.Exec(`
		DELETE FROM sessions
		WHERE id = ?
	`, sessionID)
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

func (s *Store) GetSession(sessionID string) (Session, error) {
	var (
		dataJSON  string
		ip        string
		userAgent string
		createdAt time.Time
		expiresAt time.Time
	)

	err := s.db.QueryRow(`
		SELECT data, ip, user_agent, created_at, expires_at
		FROM sessions
		WHERE id = ?
	`, sessionID).Scan(
		&dataJSON,
		&ip,
		&userAgent,
		&createdAt,
		&expiresAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}

	if err != nil {
		return Session{}, err
	}

	if time.Now().After(expiresAt) {
		return Session{}, ErrSessionExpired
	}

	var data map[string]string

	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return Session{}, err
	}

	return Session{
		Data:      data,
		IP:        ip,
		UserAgent: userAgent,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Store) UpdateSession(sessionID string, session Session) error {
	data, err := json.Marshal(session.Data)
	if err != nil {
		return err
	}

	result, err := s.db.Exec(`
		UPDATE sessions
		SET
			data = ?,
			ip = ?,
			user_agent = ?,
			created_at = ?,
			expires_at = ?
		WHERE id = ?
	`,
		data,
		session.IP,
		session.UserAgent,
		session.CreatedAt,
		session.ExpiresAt,
		sessionID,
	)
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

func (s *Store) RefreshSession(sessionID string) (Session, error) {
	var (
		dataJSON  string
		ip        string
		userAgent string
		createdAt time.Time
		expiresAt time.Time
	)

	err := s.db.QueryRow(`
		SELECT data, ip, user_agent, created_at, expires_at
		FROM sessions
		WHERE id = ?
	`, sessionID).Scan(
		&dataJSON,
		&ip,
		&userAgent,
		&createdAt,
		&expiresAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}

	if err != nil {
		return Session{}, err
	}

	if time.Now().After(expiresAt) {
		return Session{}, ErrSessionExpired
	}

	newExpiresAt := time.Now().Add(sessionDuration)

	_, err = s.db.Exec(`
		UPDATE sessions
		SET expires_at = ?
		WHERE id = ?
	`, newExpiresAt, sessionID)
	if err != nil {
		return Session{}, err
	}

	var data map[string]string

	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return Session{}, err
	}

	return Session{
		Data:      data,
		IP:        ip,
		UserAgent: userAgent,
		CreatedAt: createdAt,
		ExpiresAt: newExpiresAt,
	}, nil
}

func (s *Store) RegenerateSessionID(sessionID string) (string, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var (
		dataJSON  string
		ip        string
		userAgent string
		createdAt time.Time
		expiresAt time.Time
	)

	err = tx.QueryRow(`
		SELECT data, ip, user_agent, created_at, expires_at
		FROM sessions
		WHERE id = ?
	`, sessionID).Scan(
		&dataJSON,
		&ip,
		&userAgent,
		&createdAt,
		&expiresAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrSessionNotFound
	}

	if err != nil {
		return "", err
	}

	if time.Now().After(expiresAt) {
		return "", ErrSessionExpired
	}

	now := time.Now()
	newID := newSessionID()

	_, err = tx.Exec(`
		INSERT INTO sessions (
			id,
			data,
			ip,
			user_agent,
			created_at,
			expires_at
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		newID,
		dataJSON,
		ip,
		userAgent,
		now,
		now.Add(sessionDuration),
	)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(`
		DELETE FROM sessions
		WHERE id = ?
	`, sessionID)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return newID, nil
}

func (s *Store) NeedsRegeneration(sessionID string) (bool, error) {
	var expiresAt, createdAt time.Time

	err := s.db.QueryRow(`
		SELECT created_at, expires_at
		FROM sessions
		WHERE id = ?
	`, sessionID).Scan(&createdAt, &expiresAt)

	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrSessionNotFound
	}

	if err != nil {
		return false, err
	}

	if time.Now().After(expiresAt) {
		return false, ErrSessionExpired
	}

	return time.Since(createdAt) >= sessionAbsoluteDuration, nil
}

func (s *Store) MatchesRequest(sessionID string, r *http.Request) (bool, error) {
	var (
		ip        string
		userAgent string
		expiresAt time.Time
	)

	err := s.db.QueryRow(`
		SELECT ip, user_agent, expires_at
		FROM sessions
		WHERE id = ?
	`, sessionID).Scan(
		&ip,
		&userAgent,
		&expiresAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrSessionNotFound
	}

	if err != nil {
		return false, err
	}

	if time.Now().After(expiresAt) {
		return false, ErrSessionExpired
	}

	return ip == getClientIP(r) &&
		userAgent == r.UserAgent(), nil
}

func (s *Store) createSession(w http.ResponseWriter, r *http.Request, config CookieConfig) (string, error) {
	sessionID, err := s.AddSession(r)
	if err != nil {
		return "", err
	}

	SetSessionCookie(w, config, sessionID)

	return sessionID, nil
}

func GetSession(r *http.Request) (*Session, bool) {
	sess, ok := r.Context().Value(contextKey{}).(*Session)
	return sess, ok
}

// endregion store

// region helpers

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

func GetSessionID(r *http.Request) (string, error) {
	cookie, err := r.Cookie("session_id")
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
