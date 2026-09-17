package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
	"uuid"
)

func setupTestDB(t *testing.T) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	DB = db

	if _, err := DB.Exec(createSessionsTableQuery); err != nil {
		t.Fatal(err)
	}
}

func TestAllowedMethodMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		nextCalled     bool
	}{
		{
			name:           "GET allowed",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			nextCalled:     true,
		},
		{
			name:           "POST allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusOK,
			nextCalled:     true,
		},
		{
			name:           "PUT not allowed",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
			nextCalled:     false,
		},
		{
			name:           "DELETE not allowed",
			method:         http.MethodDelete,
			expectedStatus: http.StatusMethodNotAllowed,
			nextCalled:     false,
		},
		{
			name:           "PATCH not allowed",
			method:         http.MethodPatch,
			expectedStatus: http.StatusMethodNotAllowed,
			nextCalled:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := AllowedMethodMiddleware(next)

			req := httptest.NewRequest(tt.method, "/", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					rec.Code,
				)
			}

			if nextCalled != tt.nextCalled {
				t.Fatalf(
					"expected nextCalled=%v, got %v",
					tt.nextCalled,
					nextCalled,
				)
			}
		})
	}
}
func TestSessionMiddleware_GETCreatesSession(t *testing.T) {
	setupTestDB(t)

	called := false

	handler := SessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		sess, ok := GetSession(r)
		if !ok {
			t.Fatal("session missing from context")
		}

		if sess == nil {
			t.Fatal("session is nil")
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}

	var sessionCookie *http.Cookie

	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("expected session cookie")
	}

	if sessionCookie.Value == "" {
		t.Fatal("expected non-empty session cookie")
	}
}

func TestSessionMiddleware_POSTWithoutSession(t *testing.T) {
	setupTestDB(t)

	called := false

	handler := SessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected handler not to be called")
	}
}

func TestSessionMiddleware_GETLoadsExistingSession(t *testing.T) {
	setupTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	sess, err := createSession(rec, req)
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}

	var cookie *http.Cookie

	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookieName {
			cookie = c
			break
		}
	}

	if cookie == nil {
		t.Fatal("expected session cookie")
	}

	called := false

	handler := SessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		loaded, ok := GetSession(r)
		if !ok {
			t.Fatal("session missing from context")
		}

		if loaded.id != sess.id {
			t.Fatalf(
				"expected session %q, got %q",
				sess.id,
				loaded.id,
			)
		}

		w.WriteHeader(http.StatusOK)
	}))

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)

	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestSessionMiddleware_POSTLoadsExistingSession(t *testing.T) {
	setupTestDB(t)

	// Create existing session.
	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getRec := httptest.NewRecorder()

	sess, err := createSession(getRec, getReq)
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}

	var cookie *http.Cookie

	for _, c := range getRec.Result().Cookies() {
		if c.Name == sessionCookieName {
			cookie = c
			break
		}
	}

	if cookie == nil {
		t.Fatal("expected session cookie")
	}

	// POST with existing session.
	called := false

	handler := SessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		loaded, ok := GetSession(r)
		if !ok {
			t.Fatal("session missing from context")
		}

		if loaded.id != sess.id {
			t.Fatalf(
				"expected session %q, got %q",
				sess.id,
				loaded.id,
			)
		}

		w.WriteHeader(http.StatusOK)
	}))

	postReq := httptest.NewRequest(http.MethodPost, "/", nil)
	postReq.AddCookie(cookie)

	postRec := httptest.NewRecorder()

	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", postRec.Code)
	}

	if !called {
		t.Fatal("expected handler to be called")
	}
}
func TestSessionMiddleware_GETWithNonexistentSession(t *testing.T) {
	setupTestDB(t)

	// Cookie koji ne odgovara nijednoj sesiji u DB-u.
	oldCookie := &http.Cookie{
		Name:  sessionCookieName,
		Value: "nonexistent-session-id",
	}

	called := false

	handler := SessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		sess, ok := GetSession(r)
		if !ok {
			t.Fatal("session missing from context")
		}

		if sess == nil {
			t.Fatal("session is nil")
		}

		if sess.id == oldCookie.Value {
			t.Fatal("expected a new session")
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(oldCookie)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected handler to be called")
	}

	var newCookie *http.Cookie

	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == sessionCookieName && cookie.Value != oldCookie.Value {
			newCookie = cookie
			break
		}
	}

	if newCookie == nil {
		t.Fatal("expected new session cookie")
	}
}
func TestSessionMiddleware_POSTWithNonexistentSession(t *testing.T) {
	setupTestDB(t)

	// Cookie koji ne odgovara nijednoj sesiji u DB-u.
	cookie := &http.Cookie{
		Name:  sessionCookieName,
		Value: "nonexistent-session-id",
	}

	called := false

	handler := SessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(cookie)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected handler not to be called")
	}
}

func TestValidateSessionMiddleware_NoSession(t *testing.T) {
	handler := ValidateSessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	setupTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess := NewAnonSession(req)

	if err := sess.Save(); err != nil {
		t.Fatalf("save session: %v", err)
	}

	req = req.WithContext(
		context.WithValue(req.Context(), contextKey{}, sess),
	)

	called := false

	handler := ValidateSessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	setupTestDB(t)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	sess := NewAnonSession(req)

	if err := sess.Save(); err != nil {
		t.Fatalf("save session: %v", err)
	}

	req = req.WithContext(
		context.WithValue(req.Context(), contextKey{}, sess),
	)

	called := false

	handler := ValidateSessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	setupTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess := NewAnonSession(req)

	sess.expiresAt = time.Now().Add(-time.Minute)

	if err := sess.Save(); err != nil {
		t.Fatalf("save session: %v", err)
	}

	oldID := sess.id

	req = req.WithContext(
		context.WithValue(req.Context(), contextKey{}, sess),
	)

	called := false

	handler := ValidateSessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		loaded, ok := GetSession(r)
		if !ok {
			t.Fatal("session missing from context")
		}

		if loaded.id == oldID {
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
	setupTestDB(t)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	sess := NewAnonSession(req)

	sess.expiresAt = time.Now().Add(-time.Minute)

	if err := sess.Save(); err != nil {
		t.Fatalf("save session: %v", err)
	}

	req = req.WithContext(
		context.WithValue(req.Context(), contextKey{}, sess),
	)

	called := false

	handler := ValidateSessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	setupTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess := NewAnonSession(req)

	sess.userIP = "1.2.3.4"

	if err := sess.Save(); err != nil {
		t.Fatalf("save session: %v", err)
	}

	oldID := sess.id

	req = req.WithContext(
		context.WithValue(req.Context(), contextKey{}, sess),
	)

	called := false

	handler := ValidateSessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		loaded, ok := GetSession(r)
		if !ok {
			t.Fatal("session missing from context")
		}

		if loaded.id == oldID {
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
	setupTestDB(t)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	sess := NewAnonSession(req)

	sess.userIP = "1.2.3.4"

	if err := sess.Save(); err != nil {
		t.Fatalf("save session: %v", err)
	}

	req = req.WithContext(
		context.WithValue(req.Context(), contextKey{}, sess),
	)

	called := false

	handler := ValidateSessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	setupTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "test-agent-1")

	sess := NewAnonSession(req)
	sess.userAgent = "test-agent-2"

	if err := sess.Save(); err != nil {
		t.Fatalf("save session: %v", err)
	}

	oldID := sess.id

	req = req.WithContext(
		context.WithValue(req.Context(), contextKey{}, sess),
	)

	called := false

	handler := ValidateSessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		loaded, ok := GetSession(r)
		if !ok {
			t.Fatal("session missing from context")
		}

		if loaded.id == oldID {
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
	setupTestDB(t)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("User-Agent", "test-agent-1")

	sess := NewAnonSession(req)
	sess.userAgent = "test-agent-2"

	if err := sess.Save(); err != nil {
		t.Fatalf("save session: %v", err)
	}

	req = req.WithContext(
		context.WithValue(req.Context(), contextKey{}, sess),
	)

	called := false

	handler := ValidateSessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	setupTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess := NewAnonSession(req)

	oldExpiration := time.Now().Add(sessionDuration / 4)
	sess.expiresAt = oldExpiration

	if err := sess.Save(); err != nil {
		t.Fatalf("save session: %v", err)
	}

	req = req.WithContext(
		context.WithValue(req.Context(), contextKey{}, sess),
	)

	handler := ValidateSessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !sess.expiresAt.After(oldExpiration) {
		t.Fatalf(
			"expected expiration to be refreshed: old=%v new=%v",
			oldExpiration,
			sess.expiresAt,
		)
	}
}
func TestValidateSessionMiddleware_GETRecreatesSessionCookie(t *testing.T) {
	setupTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess := NewAnonSession(req)

	sess.expiresAt = time.Now().Add(-time.Minute)

	if err := sess.Save(); err != nil {
		t.Fatalf("save session: %v", err)
	}

	oldID := sess.id

	req = req.WithContext(
		context.WithValue(req.Context(), contextKey{}, sess),
	)

	handler := ValidateSessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var newCookie *http.Cookie

	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == sessionCookieName &&
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

// SessionMiddleware
// ├── GET bez cookie              → create
// ├── GET validan cookie          → load
// ├── GET nonexistent session     → create
// ├── POST bez cookie             → 403
// ├── POST validan cookie         → load
// └── POST nonexistent session    → 403

// ValidateSessionMiddleware
// ├── nema session u contextu     → 500
// ├── valid GET                   → next
// ├── valid POST                  → next
// ├── expired GET                 → recreate → next
// ├── expired POST                → 403
// ├── IP mismatch GET             → recreate → next
// ├── IP mismatch POST            → 403
// ├── UA mismatch GET             → recreate → next
// ├── UA mismatch POST            → 403
// ├── near expiration             → Touch
// └── recreate GET                → novi session cookie

func TestCSRFMiddleware_GET(t *testing.T) {
	setupTestDB(t)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	CSRFMiddleware(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func TestCSRFMiddleware_POSTWithoutSession(t *testing.T) {
	setupTestDB(t)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	CSRFMiddleware(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestCSRFMiddleware_POSTWithValidCSRFToken(t *testing.T) {
	setupTestDB(t)

	sess := NewAnonSession(
		httptest.NewRequest(http.MethodPost, "/", nil),
	)

	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set(csrfFieldName, sess.csrfToken)

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := context.WithValue(req.Context(), contextKey{}, sess)
	req = req.WithContext(ctx)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()

	CSRFMiddleware(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func TestCSRFMiddleware_POSTWithInvalidCSRFToken(t *testing.T) {
	setupTestDB(t)

	sess := NewAnonSession(
		httptest.NewRequest(http.MethodPost, "/", nil),
	)

	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set(csrfFieldName, "invalid-token")

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := context.WithValue(req.Context(), contextKey{}, sess)
	req = req.WithContext(ctx)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	rec := httptest.NewRecorder()

	CSRFMiddleware(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestCSRFMiddleware_POSTWithoutCSRFToken(t *testing.T) {
	setupTestDB(t)

	sess := NewAnonSession(
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

	ctx := context.WithValue(req.Context(), contextKey{}, sess)
	req = req.WithContext(ctx)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	rec := httptest.NewRecorder()

	CSRFMiddleware(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

// CSRFMiddleware
// ├── GET
// │   └── → next
// ├── POST bez session-a
// │   └── → 500
// ├── POST validan CSRF token
// │   └── → next
// ├── POST nevalidan CSRF token
// │   └── → 403
// └── POST bez CSRF tokena
//     └── → 403

func TestAuthMiddleware_NoSession(t *testing.T) {
	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	AuthMiddleware(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestAuthMiddleware_AnonymousSession(t *testing.T) {
	sess := NewAnonSession(
		httptest.NewRequest(http.MethodGet, "/", nil),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	ctx := context.WithValue(req.Context(), contextKey{}, sess)
	req = req.WithContext(ctx)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	rec := httptest.NewRecorder()

	AuthMiddleware(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}

	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestAuthMiddleware_AuthenticatedSession(t *testing.T) {
	userID := uuid.New()

	sess := NewAuthSession(
		userID,
		httptest.NewRequest(http.MethodGet, "/", nil),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	ctx := context.WithValue(req.Context(), contextKey{}, sess)
	req = req.WithContext(ctx)

	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()

	AuthMiddleware(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected next handler to be called")
	}
}
