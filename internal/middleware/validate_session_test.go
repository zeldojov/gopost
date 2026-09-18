package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zeldojov/gopost/internal/session"
)

func TestValidateSessionMiddleware_NoSession(t *testing.T) {
	handler := ValidateSession(nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler must not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestValidateSessionMiddleware_ValidGET(t *testing.T) {
	st := setupTestStore(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess := session.NewAnonSession(req)

	if err := st.SaveSession(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	req = req.WithContext(
		context.WithValue(req.Context(), session.ContextKey{}, sess),
	)

	called := false

	handler := ValidateSession(st, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestValidateSessionMiddleware_ValidPOST(t *testing.T) {
	st := setupTestStore(t)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	sess := session.NewAnonSession(req)

	if err := st.SaveSession(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	req = req.WithContext(
		context.WithValue(req.Context(), session.ContextKey{}, sess),
	)

	called := false

	handler := ValidateSession(st, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestValidateSessionMiddleware_GETRecreatesExpiredSession(t *testing.T) {
	st := setupTestStore(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess := session.NewAnonSession(req)

	sess.SetExpiresAt(time.Now().Add(-time.Minute))

	if err := st.SaveSession(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	oldID := sess.ID()

	req = req.WithContext(
		context.WithValue(req.Context(), session.ContextKey{}, sess),
	)

	called := false

	handler := ValidateSession(st, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		loaded, ok := session.GetSession(r)
		if !ok {
			t.Fatal("session missing from context")
		}

		if loaded.ID() == oldID {
			t.Fatal("expected session to be recreated")
		}

		if loaded.IsExpired() {
			t.Fatal("expected recreated session to be valid")
		}

		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestValidateSessionMiddleware_POSTRejectsExpiredSession(t *testing.T) {
	st := setupTestStore(t)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	sess := session.NewAnonSession(req)

	sess.SetExpiresAt(time.Now().Add(-time.Minute))

	if err := st.SaveSession(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	req = req.WithContext(
		context.WithValue(req.Context(), session.ContextKey{}, sess),
	)

	called := false

	handler := ValidateSession(st, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected handler not to be called")
	}
}

func TestValidateSessionMiddleware_GETRecreatesOnIPMismatch(t *testing.T) {
	st := setupTestStore(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess := session.NewAnonSession(req)

	sess.SetUserIP("1.2.3.4")

	if err := st.SaveSession(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	oldID := sess.ID()

	req = req.WithContext(
		context.WithValue(req.Context(), session.ContextKey{}, sess),
	)

	called := false

	handler := ValidateSession(st, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		loaded, ok := session.GetSession(r)
		if !ok {
			t.Fatal("session missing from context")
		}

		if loaded.ID() == oldID {
			t.Fatal("expected session to be recreated")
		}

		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestValidateSessionMiddleware_POSTRejectsIPMismatch(t *testing.T) {
	st := setupTestStore(t)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	sess := session.NewAnonSession(req)

	sess.SetUserIP("1.2.3.4")

	if err := st.SaveSession(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	req = req.WithContext(
		context.WithValue(req.Context(), session.ContextKey{}, sess),
	)

	called := false

	handler := ValidateSession(st, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected handler not to be called")
	}
}

func TestValidateSessionMiddleware_GETRecreatesOnUserAgentMismatch(t *testing.T) {
	st := setupTestStore(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "test-agent-1")

	sess := session.NewAnonSession(req)
	sess.SetUserAgent("test-agent-2")

	if err := st.SaveSession(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	oldID := sess.ID()

	req = req.WithContext(
		context.WithValue(req.Context(), session.ContextKey{}, sess),
	)

	called := false

	handler := ValidateSession(st, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		loaded, ok := session.GetSession(r)
		if !ok {
			t.Fatal("session missing from context")
		}

		if loaded.ID() == oldID {
			t.Fatal("expected session to be recreated")
		}

		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestValidateSessionMiddleware_POSTRejectsUserAgentMismatch(t *testing.T) {
	st := setupTestStore(t)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("User-Agent", "test-agent-1")

	sess := session.NewAnonSession(req)
	sess.SetUserAgent("test-agent-2")

	if err := st.SaveSession(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	req = req.WithContext(
		context.WithValue(req.Context(), session.ContextKey{}, sess),
	)

	called := false

	handler := ValidateSession(st, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected handler not to be called")
	}
}

func TestValidateSessionMiddleware_RefreshesSession(t *testing.T) {
	st := setupTestStore(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess := session.NewAnonSession(req)

	oldExpiration := time.Now().Add(session.SessionDuration() / 4)
	sess.SetExpiresAt(oldExpiration)

	if err := st.SaveSession(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	req = req.WithContext(
		context.WithValue(req.Context(), session.ContextKey{}, sess),
	)

	handler := ValidateSession(st, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !sess.ExpiresAt().After(oldExpiration) {
		t.Fatalf(
			"expected expiration to be refreshed: old=%v new=%v",
			oldExpiration,
			sess.ExpiresAt(),
		)
	}
}

func TestValidateSessionMiddleware_GETRecreatesSessionCookie(t *testing.T) {
	st := setupTestStore(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess := session.NewAnonSession(req)

	sess.SetExpiresAt(time.Now().Add(-time.Minute))

	if err := st.SaveSession(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	oldID := sess.ID()

	req = req.WithContext(
		context.WithValue(req.Context(), session.ContextKey{}, sess),
	)

	handler := ValidateSession(st, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var newCookie *http.Cookie

	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == session.SessionCookieName() &&
			cookie.Value != "" &&
			cookie.Value != oldID {
			newCookie = cookie
			break
		}
	}

	if newCookie == nil {
		t.Fatal("expected new session cookie")
	}
}
