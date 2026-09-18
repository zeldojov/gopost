package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/zeldojov/gopost/internal/session"
)

func TestGuest_AuthenticatedSession(t *testing.T) {
	userID := uuid.New()

	sess := session.NewAuthSession(
		userID,
		httptest.NewRequest(http.MethodGet, "/", nil),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	req = req.WithContext(
		context.WithValue(req.Context(), session.ContextKey{}, sess),
	)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()

	Guest(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303, got %d", rec.Code)
	}

	if location := rec.Header().Get("Location"); location != "/user/home" {
		t.Fatalf("expected redirect to /user/home, got %q", location)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestGuest_AnonymousSession(t *testing.T) {
	sess := session.NewAnonSession(
		httptest.NewRequest(http.MethodGet, "/", nil),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	req = req.WithContext(
		context.WithValue(req.Context(), session.ContextKey{}, sess),
	)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()

	Guest(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}
}
