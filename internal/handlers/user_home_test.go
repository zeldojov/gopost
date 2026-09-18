package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"

	"github.com/zeldojov/gopost/internal/session"
	"github.com/zeldojov/gopost/internal/user"
)

func TestUserHome_Success(t *testing.T) {
	setupTestDB(t)

	newUser := user.CreateUser("testuser", "password-hash")

	if err := newUser.Save(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/user/home", nil)

	sess := session.NewAuthSession(newUser.ID, req)

	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	UserHome(rec, req)

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
	setupTestDB(t)

	userID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/user/home", nil)

	sess := session.NewAuthSession(userID, req)

	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	UserHome(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			rec.Code,
		)
	}
}
func TestUserHome_MissingSession(t *testing.T) {
	setupTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/user/home", nil)

	rec := httptest.NewRecorder()

	UserHome(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}
}
func TestUserHome_AnonymousSession(t *testing.T) {
	setupTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/user/home", nil)

	sess := session.NewAnonSession(req)

	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	UserHome(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			rec.Code,
		)
	}
}
