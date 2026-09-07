package session

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	if err := CreateSessionsTable(db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return NewStore(db)
}

// region: session

func TestSession_SetGet(t *testing.T) {
	sess := Session{
		Data: make(map[string]string),
	}

	sess.Set("user_id", "123")

	value, ok := sess.Get("user_id")
	if !ok {
		t.Fatal("expected value to exist")
	}

	if value != "123" {
		t.Fatalf("expected value %q, got %q", "123", value)
	}
}

func TestSession_Delete(t *testing.T) {
	sess := Session{
		Data: make(map[string]string),
	}

	sess.Set("user_id", "123")
	sess.Delete("user_id")

	_, ok := sess.Get("user_id")
	if ok {
		t.Fatal("expected user_id to be deleted")
	}
}

func TestSession_Has(t *testing.T) {
	sess := Session{
		Data: make(map[string]string),
	}

	sess.Set("user_id", "123")

	if !sess.Has("user_id") {
		t.Fatal("expected user_id to exist")
	}

	if sess.Has("missing") {
		t.Fatal("expected missing key to not exist")
	}
}

func TestNewSession(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	req.Header.Set("User-Agent", "test-agent")

	sess := newSession(req)

	if sess.Data == nil {
		t.Fatal("expected Data to be initialized")
	}

	if sess.IP != "127.0.0.1" {
		t.Fatalf("expected IP %q, got %q", "127.0.0.1", sess.IP)
	}

	if sess.UserAgent != "test-agent" {
		t.Fatalf("expected UserAgent %q, got %q", "test-agent", sess.UserAgent)
	}

	if sess.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}

	if sess.ExpiresAt.IsZero() {
		t.Fatal("expected ExpiresAt to be set")
	}

	if !sess.ExpiresAt.After(sess.CreatedAt) {
		t.Fatal("expected ExpiresAt to be after CreatedAt")
	}
}

// endregion: session

// region: store

func TestNewStore(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	if err := CreateSessionsTable(db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store := NewStore(db)

	if store == nil {
		t.Fatal("expected store, got nil")
	}

	if store.db != db {
		t.Fatal("expected store to use provided database")
	}
}

func TestStore_AddSession(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	if err := CreateSessionsTable(db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	store := NewStore(db)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "test-agent")

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sessionID == "" {
		t.Fatal("expected session ID")
	}
}

func TestStore_GetSession(t *testing.T) {
	store := newTestStore(t)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	req.Header.Set("User-Agent", "test-agent")

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sess, err := store.GetSession(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sess.IP != "127.0.0.1" {
		t.Fatalf("expected IP %q, got %q", "127.0.0.1", sess.IP)
	}

	if sess.UserAgent != "test-agent" {
		t.Fatalf("expected user agent %q, got %q", "test-agent", sess.UserAgent)
	}

	if sess.Data == nil {
		t.Fatal("expected initialized session data")
	}
}

func TestStore_GetSession_NotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.GetSession("non-existent")

	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestStore_UpdateSession(t *testing.T) {
	store := newTestStore(t)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sess, err := store.GetSession(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sess.Set("user_id", "123")

	err = store.UpdateSession(sessionID, sess)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, err := store.GetSession(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	value, ok := updated.Get("user_id")
	if !ok {
		t.Fatal("expected user_id in session")
	}

	if value != "123" {
		t.Fatalf("expected user_id %q, got %q", "123", value)
	}
}

func TestStore_UpdateSession_NotFound(t *testing.T) {
	store := newTestStore(t)

	sess := Session{
		Data: make(map[string]string),
	}

	err := store.UpdateSession("non-existent", sess)

	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestStore_RemoveSession(t *testing.T) {
	store := newTestStore(t)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = store.RemoveSession(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = store.GetSession(sessionID)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestStore_RemoveSession_NotFound(t *testing.T) {
	store := newTestStore(t)

	err := store.RemoveSession("non-existent")

	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestStore_GetSession_Expired(t *testing.T) {
	store := newTestStore(t)

	req := httptest.NewRequest("GET", "/", nil)

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = store.db.Exec(`
		UPDATE sessions
		SET expires_at = ?
		WHERE id = ?
	`, time.Now().Add(-1*time.Hour), sessionID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = store.GetSession(sessionID)

	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired, got %v", err)
	}
}

func TestStore_RefreshSession(t *testing.T) {
	store := newTestStore(t)

	req := httptest.NewRequest("GET", "/", nil)
	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sess, err := store.GetSession(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	oldExpiresAt := sess.ExpiresAt

	time.Sleep(time.Millisecond)

	refreshed, err := store.RefreshSession(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !refreshed.ExpiresAt.After(oldExpiresAt) {
		t.Fatalf("expected ExpiresAt to be extended")
	}
}

func TestStore_RefreshSession_Expired(t *testing.T) {
	store := newTestStore(t)

	req := httptest.NewRequest("GET", "/", nil)

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = store.db.Exec(`
		UPDATE sessions
		SET expires_at = ?
		WHERE id = ?
	`, time.Now().Add(-1*time.Hour), sessionID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = store.RefreshSession(sessionID)

	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired, got %v", err)
	}
}

func TestStore_RefreshSession_NotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.RefreshSession("non-existent")

	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestStore_RegenerateSessionID(t *testing.T) {
	store := newTestStore(t)

	req := httptest.NewRequest("GET", "/", nil)
	oldSessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newSessionID, err := store.RegenerateSessionID(oldSessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if newSessionID == oldSessionID {
		t.Fatal("expected new session ID")
	}
}

func TestStore_RegenerateSessionID_PreservesSession(t *testing.T) {
	store := newTestStore(t)

	req := httptest.NewRequest("GET", "/", nil)
	oldSessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	oldSession, err := store.GetSession(oldSessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	oldSession.Set("user_id", "123")
	oldSession.CreatedAt = time.Now().Add(-8 * 24 * time.Hour)
	oldSession.ExpiresAt = time.Now().Add(10 * time.Minute)

	err = store.UpdateSession(oldSessionID, oldSession)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	oldSession, err = store.GetSession(oldSessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newSessionID, err := store.RegenerateSessionID(oldSessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newSession, err := store.GetSession(newSessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !newSession.CreatedAt.After(oldSession.CreatedAt) {
		t.Fatal("expected CreatedAt to be regenerated")
	}

	if !newSession.ExpiresAt.After(oldSession.ExpiresAt) {
		t.Fatal("expected ExpiresAt to be regenerated")
	}

	value, ok := newSession.Get("user_id")
	if !ok {
		t.Fatal("expected session data to be preserved")
	}

	if value != "123" {
		t.Fatalf("expected user_id %q, got %q", "123", value)
	}

	if newSession.IP != oldSession.IP {
		t.Fatal("expected IP to be preserved")
	}

	if newSession.UserAgent != oldSession.UserAgent {
		t.Fatal("expected UserAgent to be preserved")
	}

	if _, err := store.GetSession(oldSessionID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected old session ID to be removed, got %v", err)
	}
}

func TestStore_RegenerateSessionID_Expired(t *testing.T) {
	store := newTestStore(t)

	req := httptest.NewRequest("GET", "/", nil)

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = store.db.Exec(`
		UPDATE sessions
		SET expires_at = ?
		WHERE id = ?
	`, time.Now().Add(-time.Hour), sessionID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = store.RegenerateSessionID(sessionID)

	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired, got %v", err)
	}
}

func TestStore_RegenerateSessionID_NotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.RegenerateSessionID("non-existent")

	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestStore_NeedsRegeneration(t *testing.T) {
	store := newTestStore(t)

	req := httptest.NewRequest("GET", "/", nil)
	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	needsRegeneration, err := store.NeedsRegeneration(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if needsRegeneration {
		t.Fatal("expected session not to need regeneration")
	}
}

func TestStore_NeedsRegeneration_Required(t *testing.T) {
	store := newTestStore(t)

	req := httptest.NewRequest("GET", "/", nil)

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = store.db.Exec(`
		UPDATE sessions
		SET
			created_at = ?,
			expires_at = ?
		WHERE id = ?
	`,
		time.Now().Add(-8*24*time.Hour),
		time.Now().Add(time.Hour),
		sessionID,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	needsRegeneration, err := store.NeedsRegeneration(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !needsRegeneration {
		t.Fatal("expected session to need regeneration")
	}
}

func TestStore_NeedsRegeneration_Expired(t *testing.T) {
	store := newTestStore(t)

	req := httptest.NewRequest("GET", "/", nil)

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = store.db.Exec(`
		UPDATE sessions
		SET
			created_at = ?,
			expires_at = ?
		WHERE id = ?
	`,
		time.Now().Add(-8*24*time.Hour),
		time.Now().Add(-time.Hour),
		sessionID,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = store.NeedsRegeneration(sessionID)

	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired, got %v", err)
	}
}

func TestStore_NeedsRegeneration_NotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.NeedsRegeneration("non-existent")

	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestStore_MatchesRequest(t *testing.T) {
	store := newTestStore(t)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	req.Header.Set("User-Agent", "test-agent")

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	matches, err := store.MatchesRequest(sessionID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !matches {
		t.Fatal("expected session to match request")
	}
}

func TestStore_MatchesRequest_IPMismatch(t *testing.T) {
	store := newTestStore(t)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	req.Header.Set("User-Agent", "test-agent")

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req.RemoteAddr = "192.168.1.100:8080"

	matches, err := store.MatchesRequest(sessionID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if matches {
		t.Fatal("expected session not to match request")
	}
}

func TestStore_MatchesRequest_UserAgentMismatch(t *testing.T) {
	store := newTestStore(t)

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	req.Header.Set("User-Agent", "test-agent")

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req.Header.Set("User-Agent", "different-agent")

	matches, err := store.MatchesRequest(sessionID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if matches {
		t.Fatal("expected session not to match request")
	}
}

// endregion: store

// region: helpers

func TestGetSessionID(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: "abc123",
	})

	sessionID, err := GetSessionID(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sessionID != "abc123" {
		t.Fatalf("expected session ID %q, got %q", "abc123", sessionID)
	}
}

func TestGetSessionID_CookieNotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	_, err := GetSessionID(req)

	if !errors.Is(err, ErrSessionCookieNotFound) {
		t.Fatalf("expected ErrSessionCookieNotFound, got %v", err)
	}
}

func TestSetSessionCookie(t *testing.T) {
	config := CookieConfig{
		Name:     "session_id",
		Path:     "/",
		Secure:   true,
		HTTPOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	rec := httptest.NewRecorder()

	SetSessionCookie(rec, config, "abc123")

	result := rec.Result()
	cookies := result.Cookies()

	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]

	if cookie.Name != config.Name {
		t.Fatalf("expected cookie name %q, got %q", config.Name, cookie.Name)
	}

	if cookie.Value != "abc123" {
		t.Fatalf("expected cookie value %q, got %q", "abc123", cookie.Value)
	}

	if cookie.Path != config.Path {
		t.Fatalf("expected cookie path %q, got %q", config.Path, cookie.Path)
	}

	if cookie.HttpOnly != config.HTTPOnly {
		t.Fatalf("expected HttpOnly %v, got %v", config.HTTPOnly, cookie.HttpOnly)
	}

	if cookie.Secure != config.Secure {
		t.Fatalf("expected Secure %v, got %v", config.Secure, cookie.Secure)
	}

	if cookie.SameSite != config.SameSite {
		t.Fatalf(
			"expected SameSite %v, got %v",
			config.SameSite,
			cookie.SameSite,
		)
	}
}

func TestDeleteSessionCookie(t *testing.T) {
	rec := httptest.NewRecorder()

	config := CookieConfig{
		Name:     "session_id",
		Path:     "/",
		Secure:   true,
		HTTPOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	DeleteSessionCookie(rec, config)

	result := rec.Result()
	cookies := result.Cookies()

	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]

	if cookie.Name != "session_id" {
		t.Fatalf("expected cookie name %q, got %q", "session_id", cookie.Name)
	}

	if cookie.Value != "" {
		t.Fatalf("expected empty cookie value, got %q", cookie.Value)
	}

	if cookie.Path != "/" {
		t.Fatalf("expected cookie path %q, got %q", "/", cookie.Path)
	}

	if cookie.MaxAge != -1 {
		t.Fatalf("expected MaxAge -1, got %d", cookie.MaxAge)
	}

	if !cookie.HttpOnly {
		t.Fatal("expected HttpOnly cookie")
	}

	if !cookie.Secure {
		t.Fatal("expected Secure cookie")
	}

	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf(
			"expected SameSite %v, got %v",
			http.SameSiteLaxMode,
			cookie.SameSite,
		)
	}
}

// endregion: helpers

// region: CSRF

func TestSession_GetCSRFToken(t *testing.T) {
	sess := Session{
		Data: make(map[string]string),
	}

	token := sess.GetCSRFToken()

	if token == "" {
		t.Fatal("expected CSRF token")
	}

	storedToken, ok := sess.Get("csrf_token")
	if !ok {
		t.Fatal("expected CSRF token to be stored in session")
	}

	if storedToken != token {
		t.Fatalf("expected stored token %q, got %q", token, storedToken)
	}

	secondToken := sess.GetCSRFToken()

	if secondToken != token {
		t.Fatalf("expected existing token %q, got %q", token, secondToken)
	}
}

// endregion: CSRF
