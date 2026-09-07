package database

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestSessionRepository_CreateAndGetSession(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	if err := CreateSessionsTable(db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repository := NewSessionRepository(db)

	createdAt := time.Now().Truncate(time.Microsecond)
	expiresAt := createdAt.Add(30 * time.Minute)

	expected := Session{
		ID: "session-123",
		Data: map[string]string{
			"user_id": "42",
			"role":    "admin",
		},
		IP:        "127.0.0.1",
		UserAgent: "test-agent",
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}

	if err := repository.CreateSession(expected); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	actual, err := repository.GetSession(expected.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if actual.ID != expected.ID {
		t.Fatalf("expected ID %q, got %q", expected.ID, actual.ID)
	}

	if !reflect.DeepEqual(actual.Data, expected.Data) {
		t.Fatalf("expected data %#v, got %#v", expected.Data, actual.Data)
	}

	if actual.IP != expected.IP {
		t.Fatalf("expected IP %q, got %q", expected.IP, actual.IP)
	}

	if actual.UserAgent != expected.UserAgent {
		t.Fatalf(
			"expected user agent %q, got %q",
			expected.UserAgent,
			actual.UserAgent,
		)
	}

	if !actual.CreatedAt.Equal(expected.CreatedAt) {
		t.Fatalf(
			"expected created at %v, got %v",
			expected.CreatedAt,
			actual.CreatedAt,
		)
	}

	if !actual.ExpiresAt.Equal(expected.ExpiresAt) {
		t.Fatalf(
			"expected expires at %v, got %v",
			expected.ExpiresAt,
			actual.ExpiresAt,
		)
	}
}
func TestSessionRepository_GetSession_NotFound(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	if err := CreateSessionsTable(db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repository := NewSessionRepository(db)

	_, err = repository.GetSession("missing-session")

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
func TestSessionRepository_UpdateSession(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	if err := CreateSessionsTable(db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repository := NewSessionRepository(db)

	createdAt := time.Now().Truncate(time.Microsecond)
	expiresAt := createdAt.Add(30 * time.Minute)

	session := Session{
		ID: "session-123",
		Data: map[string]string{
			"user_id": "42",
		},
		IP:        "127.0.0.1",
		UserAgent: "test-agent",
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}

	if err := repository.CreateSession(session); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	session.Data["role"] = "admin"
	session.IP = "192.168.1.10"
	session.UserAgent = "updated-agent"

	if err := repository.UpdateSession(session); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	actual, err := repository.GetSession(session.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(actual.Data, session.Data) {
		t.Fatalf("expected data %#v, got %#v", session.Data, actual.Data)
	}

	if actual.IP != session.IP {
		t.Fatalf("expected IP %q, got %q", session.IP, actual.IP)
	}

	if actual.UserAgent != session.UserAgent {
		t.Fatalf(
			"expected user agent %q, got %q",
			session.UserAgent,
			actual.UserAgent,
		)
	}
}
func TestSessionRepository_UpdateSession_NotFound(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	if err := CreateSessionsTable(db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repository := NewSessionRepository(db)

	session := Session{
		ID: "missing-session",
		Data: map[string]string{
			"user_id": "42",
		},
		IP:        "127.0.0.1",
		UserAgent: "test-agent",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	err = repository.UpdateSession(session)

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
func TestSessionRepository_DeleteSession(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	if err := CreateSessionsTable(db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repository := NewSessionRepository(db)

	session := Session{
		ID: "session-123",
		Data: map[string]string{
			"user_id": "42",
		},
		IP:        "127.0.0.1",
		UserAgent: "test-agent",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	if err := repository.CreateSession(session); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := repository.DeleteSession(session.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = repository.GetSession(session.ID)

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
func TestSessionRepository_DeleteSession_NotFound(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	if err := CreateSessionsTable(db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repository := NewSessionRepository(db)

	err = repository.DeleteSession("missing-session")

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
func TestSessionRepository_RegenerateSession(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	if err := CreateSessionsTable(db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repository := NewSessionRepository(db)

	oldSession := Session{
		ID: "old-session",
		Data: map[string]string{
			"user_id": "42",
		},
		IP:        "127.0.0.1",
		UserAgent: "test-agent",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	if err := repository.CreateSession(oldSession); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newSession := Session{
		ID: "new-session",
		Data: map[string]string{
			"user_id": "42",
			"role":    "admin",
		},
		IP:        "127.0.0.1",
		UserAgent: "test-agent",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	if err := repository.RegenerateSession(
		oldSession.ID,
		newSession,
	); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = repository.GetSession(oldSession.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected old session to be deleted, got %v", err)
	}

	actual, err := repository.GetSession(newSession.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(actual.Data, newSession.Data) {
		t.Fatalf(
			"expected data %#v, got %#v",
			newSession.Data,
			actual.Data,
		)
	}

	if actual.IP != newSession.IP {
		t.Fatalf("expected IP %q, got %q", newSession.IP, actual.IP)
	}

	if actual.UserAgent != newSession.UserAgent {
		t.Fatalf(
			"expected user agent %q, got %q",
			newSession.UserAgent,
			actual.UserAgent,
		)
	}
}
func TestSessionRepository_RegenerateSession_NotFound(t *testing.T) {
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	if err := CreateSessionsTable(db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repository := NewSessionRepository(db)

	session := Session{
		ID: "new-session",
		Data: map[string]string{
			"user_id": "42",
		},
		IP:        "127.0.0.1",
		UserAgent: "test-agent",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	err = repository.RegenerateSession(
		"missing-session",
		session,
	)

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	_, err = repository.GetSession(session.ID)

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf(
			"expected new session not to exist after failed regeneration, got %v",
			err,
		)
	}
}
