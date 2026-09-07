package session

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

// endregion: store

// region: helpers

func TestGetSessionID(t *testing.T) {
	config := CookieConfig{
		Name: "session_id",
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  config.Name,
		Value: "abc123",
	})

	sessionID, err := GetSessionID(req, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sessionID != "abc123" {
		t.Fatalf("expected session ID %q, got %q", "abc123", sessionID)
	}
}
func TestGetSessionID_CookieNotFound(t *testing.T) {
	config := CookieConfig{
		Name: "session_id",
	}

	req := httptest.NewRequest("GET", "/", nil)

	_, err := GetSessionID(req, config)

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
