package session

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	database "github.com/zeldojov/gopost/internal/database"
)

type mockSessionRepository struct {
	createdSession database.Session
}

func (m *mockSessionRepository) CreateSession(session database.Session) error {
	m.createdSession = session
	return nil
}

func (m *mockSessionRepository) GetSession(id string) (database.Session, error) {
	if m.createdSession.ID != id {
		return database.Session{}, database.ErrNotFound
	}

	return m.createdSession, nil
}

func (m *mockSessionRepository) UpdateSession(sess database.Session) error {
	if m.createdSession.ID != sess.ID {
		return database.ErrNotFound
	}

	m.createdSession = sess
	return nil
}

func (m *mockSessionRepository) DeleteSession(id string) error {
	if m.createdSession.ID != id {
		return database.ErrNotFound
	}

	m.createdSession = database.Session{}
	return nil
}

func (m *mockSessionRepository) RegenerateSession(oldID string, sess database.Session) error {
	if m.createdSession.ID != oldID {
		return database.ErrNotFound
	}

	m.createdSession = sess
	return nil
}

func TestNewStore(t *testing.T) {
	repository := &mockSessionRepository{}

	store := NewStore(repository)

	if store == nil {
		t.Fatal("expected store, got nil")
	}

	if store.repository != repository {
		t.Fatal("expected store to use provided repository")
	}
}

func TestStore_AddSession(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "test-agent")

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sessionID == "" {
		t.Fatal("expected session ID")
	}

	if repository.createdSession.ID != sessionID {
		t.Fatal("expected repository to receive session ID")
	}

	if repository.createdSession.UserAgent != "test-agent" {
		t.Fatal("expected repository to receive user agent")
	}
}
func TestStore_GetSession(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
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
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	_, err := store.GetSession("non-existent")

	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestStore_GetSession_Expired(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repository.createdSession.ExpiresAt = time.Now().Add(-1 * time.Hour)

	_, err = store.GetSession(sessionID)

	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired, got %v", err)
	}
}

func TestStore_UpdateSession(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
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
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	sess := Session{
		Data: make(map[string]string),
	}

	err := store.UpdateSession("non-existent", sess)

	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestStore_RemoveSession(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
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
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	err := store.RemoveSession("non-existent")

	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestStore_RefreshSession(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

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
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repository.createdSession.ExpiresAt = time.Now().Add(-1 * time.Hour)

	_, err = store.RefreshSession(sessionID)

	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired, got %v", err)
	}
}
func TestStore_RefreshSession_NotFound(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	_, err := store.RefreshSession("non-existent")

	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestStore_RegenerateSessionID(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

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
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

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
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repository.createdSession.ExpiresAt = time.Now().Add(-time.Hour)

	_, err = store.RegenerateSessionID(sessionID)

	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired, got %v", err)
	}
}
func TestStore_RegenerateSessionID_NotFound(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	_, err := store.RegenerateSessionID("non-existent")

	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestStore_NeedsRegeneration(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

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
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repository.createdSession.CreatedAt = time.Now().Add(-8 * 24 * time.Hour)
	repository.createdSession.ExpiresAt = time.Now().Add(time.Hour)

	needsRegeneration, err := store.NeedsRegeneration(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !needsRegeneration {
		t.Fatal("expected session to need regeneration")
	}
}
func TestStore_NeedsRegeneration_Expired(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repository.createdSession.CreatedAt = time.Now().Add(-8 * 24 * time.Hour)
	repository.createdSession.ExpiresAt = time.Now().Add(-time.Hour)

	_, err = store.NeedsRegeneration(sessionID)

	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired, got %v", err)
	}
}
func TestStore_NeedsRegeneration_NotFound(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	_, err := store.NeedsRegeneration("non-existent")

	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestStore_MatchesRequest(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
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
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
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
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
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
