package session

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
