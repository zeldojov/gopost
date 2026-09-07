package session

import (
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
