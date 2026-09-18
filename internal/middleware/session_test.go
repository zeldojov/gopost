package middleware

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zeldojov/gopost/internal/session"
	"github.com/zeldojov/gopost/internal/store"

	_ "modernc.org/sqlite"
)

func setupTestStore(t *testing.T) *store.Store {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	st := store.NewStore(db)

	if err := st.CreateSessionsTable(); err != nil {
		t.Fatal(err)
	}

	return st
}

func TestSessionMiddleware_GETCreatesSession(t *testing.T) {
	st := setupTestStore(t)

	called := false

	handler := Session(st, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	st := setupTestStore(t)

	called := false

	handler := Session(st, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	st := setupTestStore(t)

	// Kreiraj i sačuvaj postojeću sesiju.
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	sess := session.NewAnonSession(req)

	if err := st.SaveSession(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	cookie := &http.Cookie{
		Name:  session.SessionCookieName(),
		Value: sess.ID(),
	}

	called := false

	handler := Session(st, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestSessionMiddleware_POSTLoadsExistingSession(t *testing.T) {
	st := setupTestStore(t)

	// Kreiraj postojeću sesiju.
	getReq := httptest.NewRequest(http.MethodGet, "/", nil)

	sess := session.NewAnonSession(getReq)

	if err := st.SaveSession(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	cookie := &http.Cookie{
		Name:  session.SessionCookieName(),
		Value: sess.ID(),
	}

	// POST sa postojećom sesijom.
	called := false

	handler := Session(st, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	st := setupTestStore(t)

	oldCookie := &http.Cookie{
		Name:  session.SessionCookieName(),
		Value: "nonexistent-session-id",
	}

	called := false

	handler := Session(st, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		if cookie.Name == session.SessionCookieName() &&
			cookie.Value != oldCookie.Value &&
			cookie.Value != "" {
			newCookie = cookie
			break
		}
	}

	if newCookie == nil {
		t.Fatal("expected new session cookie")
	}
}

func TestSessionMiddleware_POSTWithNonexistentSession(t *testing.T) {
	st := setupTestStore(t)

	cookie := &http.Cookie{
		Name:  session.SessionCookieName(),
		Value: "nonexistent-session-id",
	}

	called := false

	handler := Session(st, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
