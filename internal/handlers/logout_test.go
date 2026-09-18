package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/zeldojov/gopost/internal/session"
)

func TestLogout_Success(t *testing.T) {
	setupTestDB(t)

	userID := uuid.New()

	req := httptest.NewRequest(
		http.MethodPost,
		"/logout",
		nil,
	)

	sess := session.NewAuthSession(userID, req)

	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	sessionID := sess.ID()

	// Handler očekuje session u contextu.
	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	// Handler očekuje session cookie.
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName(),
		Value: sessionID,
	})

	rec := httptest.NewRecorder()

	Logout(rec, req)

	// Logout mora da redirectuje na login.
	if rec.Code != http.StatusSeeOther {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusSeeOther,
			rec.Code,
		)
	}

	if location := rec.Header().Get("Location"); location != "/login" {
		t.Fatalf(
			"expected redirect to /login, got %q",
			location,
		)
	}

	// Session mora biti obrisana iz DB-a.
	loadedSession := &session.Session{}

	err := loadedSession.Load(sessionID)

	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Fatalf(
			"expected session to be deleted, got error %v",
			err,
		)
	}

	// Cookie mora biti poništen.
	cookies := rec.Result().Cookies()

	var foundDeletedCookie bool

	for _, cookie := range cookies {
		if cookie.Name == session.SessionCookieName() {
			foundDeletedCookie = true

			if cookie.Value != "" {
				t.Fatalf(
					"expected deleted session cookie to have empty value, got %q",
					cookie.Value,
				)
			}

			if cookie.MaxAge >= 0 {
				t.Fatalf(
					"expected deleted session cookie to have negative MaxAge, got %d",
					cookie.MaxAge,
				)
			}

			break
		}
	}

	if !foundDeletedCookie {
		t.Fatal("expected session deletion cookie")
	}
}
