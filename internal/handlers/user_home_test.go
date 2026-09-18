package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"

	"github.com/zeldojov/gopost/internal/session"
	"github.com/zeldojov/gopost/internal/store"
	"github.com/zeldojov/gopost/internal/user"
)

func setupTestStore(t *testing.T) *store.Store {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	db.SetMaxOpenConns(1)

	t.Cleanup(func() {
		db.Close()
	})

	st := store.NewStore(db)

	if err := st.CreateUsersTable(); err != nil {
		t.Fatal(err)
	}

	if err := st.CreateSessionsTable(); err != nil {
		t.Fatal(err)
	}

	return st
}

func TestUserHome_Success(t *testing.T) {
	st := setupTestStore(t)

	newUser := user.CreateUser("testuser", "password-hash")

	if err := st.SaveUser(newUser); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/user/home", nil)

	sess := session.NewAuthSession(newUser.ID, req)

	if err := st.SaveSession(sess); err != nil {
		t.Fatal(err)
	}

	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	handler := NewHandler(st)

	rec := httptest.NewRecorder()

	handler.UserHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	body := rec.Body.String()

	if !strings.Contains(body, "Welcome, testuser!") {
		t.Fatal(`expected body to contain "Welcome, testuser!"`)
	}

	if !strings.Contains(body, `action="/logout"`) {
		t.Fatal(`expected logout form action "/logout"`)
	}

	if !strings.Contains(body, `method="POST"`) {
		t.Fatal(`expected logout form method "POST"`)
	}

	expectedCSRF := `<input type="hidden" name="csrf_token" value="` +
		sess.CSRFToken() +
		`">`

	if !strings.Contains(body, expectedCSRF) {
		t.Fatal("expected session CSRF token in logout form")
	}
}

func TestUserHome_UserNotFound(t *testing.T) {
	st := setupTestStore(t)

	userID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/user/home", nil)

	sess := session.NewAuthSession(userID, req)

	if err := st.SaveSession(sess); err != nil {
		t.Fatal(err)
	}

	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	handler := NewHandler(st)

	rec := httptest.NewRecorder()

	handler.UserHome(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			rec.Code,
		)
	}
}

func TestUserHome_MissingSession(t *testing.T) {
	st := setupTestStore(t)

	req := httptest.NewRequest(http.MethodGet, "/user/home", nil)

	handler := NewHandler(st)

	rec := httptest.NewRecorder()

	handler.UserHome(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}
}

func TestUserHome_AnonymousSession(t *testing.T) {
	st := setupTestStore(t)

	req := httptest.NewRequest(http.MethodGet, "/user/home", nil)

	sess := session.NewAnonSession(req)

	if err := st.SaveSession(sess); err != nil {
		t.Fatal(err)
	}

	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	handler := NewHandler(st)

	rec := httptest.NewRecorder()

	handler.UserHome(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			rec.Code,
		)
	}
}
