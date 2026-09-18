package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"uuid"

	"github.com/zeldojov/gopost/internal/session"
)

func authChain(handler http.Handler) http.Handler {
	return AllowedMethod(
		Session(
			ValidateSession(
				CSRF(
					Auth(handler),
				),
			),
		),
	)
}

func TestAuth_NoSession(t *testing.T) {
	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	Auth(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestAuth_AnonymousSession(t *testing.T) {
	sess := session.NewAnonSession(
		httptest.NewRequest(http.MethodGet, "/", nil),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	rec := httptest.NewRecorder()

	Auth(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303, got %d", rec.Code)
	}

	if location := rec.Header().Get("Location"); location != "/login" {
		t.Fatalf("expected redirect to /login, got %q", location)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestAuth_AuthenticatedSession(t *testing.T) {
	userID := uuid.New()

	sess := session.NewAuthSession(
		userID,
		httptest.NewRequest(http.MethodGet, "/", nil),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()

	Auth(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func TestAuthChain_GETAnonymous(t *testing.T) {
	setupTestDB(t)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/user/home", nil)
	rec := httptest.NewRecorder()

	authChain(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status 303, got %d", rec.Code)
	}

	if location := rec.Header().Get("Location"); location != "/login" {
		t.Fatalf("expected redirect to /login, got %q", location)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestAuthChain_POSTWithoutSession(t *testing.T) {
	setupTestDB(t)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodPost, "/user/home", nil)
	rec := httptest.NewRecorder()

	authChain(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestAuthChain_AuthenticatedGET(t *testing.T) {
	setupTestDB(t)

	userID := uuid.New()

	sess := session.NewAuthSession(
		userID,
		httptest.NewRequest(http.MethodGet, "/", nil),
	)

	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/user/home", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName(),
		Value: sess.ID(),
	})

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()

	authChain(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func TestAuthChain_AuthenticatedPOST(t *testing.T) {
	setupTestDB(t)

	userID := uuid.New()

	sess := session.NewAuthSession(
		userID,
		httptest.NewRequest(http.MethodPost, "/", nil),
	)

	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set(session.CSRFFieldName(), sess.CSRFToken())

	req := httptest.NewRequest(
		http.MethodPost,
		"/user/home",
		strings.NewReader(form.Encode()),
	)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName(),
		Value: sess.ID(),
	})

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()

	authChain(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func TestAuthChain_InvalidCSRF(t *testing.T) {
	setupTestDB(t)

	userID := uuid.New()

	sess := session.NewAuthSession(
		userID,
		httptest.NewRequest(http.MethodPost, "/", nil),
	)

	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set(session.CSRFFieldName(), "invalid-token")

	req := httptest.NewRequest(
		http.MethodPost,
		"/user/home",
		strings.NewReader(form.Encode()),
	)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName(),
		Value: sess.ID(),
	})

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	rec := httptest.NewRecorder()

	authChain(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}
