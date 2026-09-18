package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/zeldojov/gopost/internal/session"
)

func publicChain(handler http.Handler) http.Handler {
	return AllowedMethod(
		Session(
			ValidateSession(
				CSRF(handler),
			),
		),
	)
}

func TestCSRF_GET(t *testing.T) {
	setupTestDB(t)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	CSRF(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func TestCSRF_POSTWithoutSession(t *testing.T) {
	setupTestDB(t)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	CSRF(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestCSRF_POSTWithValidCSRFToken(t *testing.T) {
	setupTestDB(t)

	sess := session.NewAnonSession(
		httptest.NewRequest(http.MethodPost, "/", nil),
	)

	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set(session.CSRFFieldName(), sess.CSRFToken())

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()

	CSRF(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func TestCSRF_POSTWithInvalidCSRFToken(t *testing.T) {
	setupTestDB(t)

	sess := session.NewAnonSession(
		httptest.NewRequest(http.MethodPost, "/", nil),
	)

	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set(session.CSRFFieldName(), "invalid-token")

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	rec := httptest.NewRecorder()

	CSRF(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestCSRF_POSTWithoutCSRFToken(t *testing.T) {
	setupTestDB(t)

	sess := session.NewAnonSession(
		httptest.NewRequest(http.MethodPost, "/", nil),
	)

	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		nil,
	)

	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	rec := httptest.NewRecorder()

	CSRF(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestPublicChain_GET(t *testing.T) {
	setupTestDB(t)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	publicChain(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func TestPublicChain_POSTWithoutSession(t *testing.T) {
	setupTestDB(t)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	publicChain(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestPublicChain_POSTWithValidCSRF(t *testing.T) {
	setupTestDB(t)

	sess := session.NewAnonSession(
		httptest.NewRequest(http.MethodPost, "/", nil),
	)

	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set(session.CSRFFieldName(), sess.CSRFToken())

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
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

	publicChain(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func TestPublicChain_POSTWithInvalidCSRF(t *testing.T) {
	setupTestDB(t)

	sess := session.NewAnonSession(
		httptest.NewRequest(http.MethodPost, "/", nil),
	)

	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set(session.CSRFFieldName(), "invalid-token")

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
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

	publicChain(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}
