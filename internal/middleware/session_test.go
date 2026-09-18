package middleware

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zeldojov/gopost/internal/session"
	_ "modernc.org/sqlite"
)

var DB *sql.DB

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

	if err := session.CreateSessionsTable(DB); err != nil {
		t.Fatal(err)
	}
}

func TestSessionMiddleware_GETCreatesSession(t *testing.T) {
	setupTestDB(t)

	called := false

	handler := Session(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		sess, ok := session.GetSession(r)
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
		if cookie.Name == session.SessionCookieName() {
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

	handler := Session(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	sess, err := session.CreateSession(rec, req)
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}

	var cookie *http.Cookie

	for _, c := range rec.Result().Cookies() {
		if c.Name == session.SessionCookieName() {
			cookie = c
			break
		}
	}

	if cookie == nil {
		t.Fatal("expected session cookie")
	}

	called := false

	handler := Session(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		loaded, ok := session.GetSession(r)
		if !ok {
			t.Fatal("session missing from context")
		}

		if loaded.ID() != sess.ID() {
			t.Fatalf(
				"expected session %q, got %q",
				sess.ID(),
				loaded.ID(),
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

	sess, err := session.CreateSession(getRec, getReq)
	if err != nil {
		t.Fatalf("createSession: %v", err)
	}

	var cookie *http.Cookie

	for _, c := range getRec.Result().Cookies() {
		if c.Name == session.SessionCookieName() {
			cookie = c
			break
		}
	}

	if cookie == nil {
		t.Fatal("expected session cookie")
	}

	// POST with existing session.
	called := false

	handler := Session(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		loaded, ok := session.GetSession(r)
		if !ok {
			t.Fatal("session missing from context")
		}

		if loaded.ID() != sess.ID() {
			t.Fatalf(
				"expected session %q, got %q",
				sess.ID(),
				loaded.ID(),
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
		Name:  session.SessionCookieName(),
		Value: "nonexistent-session-id",
	}

	called := false

	handler := Session(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		sess, ok := session.GetSession(r)
		if !ok {
			t.Fatal("session missing from context")
		}

		if sess == nil {
			t.Fatal("session is nil")
		}

		if sess.ID() == oldCookie.Value {
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
		if cookie.Name == session.SessionCookieName() && cookie.Value != oldCookie.Value {
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
		Name:  session.SessionCookieName(),
		Value: "nonexistent-session-id",
	}

	called := false

	handler := Session(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
