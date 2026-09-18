package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/zeldojov/gopost/internal/session"
	"github.com/zeldojov/gopost/internal/user"
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

func parseSessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	resp := rec.Result()
	cookies := resp.Cookies()

	for _, cookie := range cookies {
		if cookie.Name == session.SessionCookieName() {
			return cookie
		}
	}

	t.Fatal("session cookie not found")
	return nil
}
func TestLogin_UserNotFound(t *testing.T) {
	setupTestDB(t)

	req := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(url.Values{
			"username": {"nonexistent"},
			"password": {"ValidPassword123!"},
		}.Encode()),
	)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	sess := session.NewAnonSession(req)

	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName(),
		Value: sess.ID(),
	})

	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	Login(rec, req)

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

	if message, ok := sess.GetFlash("error"); !ok {
		t.Fatal("expected flash error")
	} else if message != "invalid credentials" {
		t.Fatalf(
			"expected flash error %q, got %q",
			"invalid credentials",
			message,
		)
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	setupTestDB(t)

	password := "ValidPassword123!"

	passwordHash, err := user.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}

	newUser := user.CreateUser("testuser", passwordHash)

	if err := newUser.Save(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(url.Values{
			"username": {"testuser"},
			"password": {"WrongPassword123!"},
		}.Encode()),
	)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	sess := session.NewAnonSession(req)

	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName(),
		Value: sess.ID(),
	})

	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	Login(rec, req)

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

	if message, ok := sess.GetFlash("error"); !ok {
		t.Fatal("expected flash error")
	} else if message != "invalid credentials" {
		t.Fatalf(
			"expected flash error %q, got %q",
			"invalid credentials",
			message,
		)
	}
}

func TestLogin_Success(t *testing.T) {
	setupTestDB(t)

	password := "ValidPassword123!"

	passwordHash, err := user.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}

	newUser := user.CreateUser("testuser", passwordHash)

	if err := newUser.Save(); err != nil {
		t.Fatal(err)
	}

	// Kreiraj početnu anonimnu sesiju.
	initialReq := httptest.NewRequest(http.MethodGet, "/login", nil)

	sess := session.NewAnonSession(initialReq)

	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	oldSessionID := sess.ID()

	// Napravi login request.
	req := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(url.Values{
			"username": {"testuser"},
			"password": {password},
		}.Encode()),
	)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName(),
		Value: oldSessionID,
	})

	// Handler očekuje session u contextu.
	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	Login(rec, req)

	// Uspešan login mora da redirectuje.
	if rec.Code != http.StatusSeeOther {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusSeeOther,
			rec.Code,
		)
	}

	if location := rec.Header().Get("Location"); location != "/user/home" {
		t.Fatalf(
			"expected redirect to /user/home, got %q",
			location,
		)
	}

	// Authenticate() mora da regeneriše session ID.
	if sess.ID() == oldSessionID {
		t.Fatal("expected session ID to change")
	}

	// Session mora biti authenticated.
	if !sess.IsAuthenticated() {
		t.Fatal("expected session to be authenticated")
	}

	if sess.UserID() == nil {
		t.Fatal("expected user ID in session")
	}

	if *sess.UserID() != newUser.ID {
		t.Fatalf(
			"expected session user ID %q, got %q",
			newUser.ID,
			*sess.UserID(),
		)
	}

	savedSession := &session.Session{}

	if err := savedSession.Load(sess.ID()); err != nil {
		t.Fatalf(
			"failed to load authenticated session: %v",
			err,
		)
	}

	if !savedSession.IsAuthenticated() {
		t.Fatal("expected saved session to be authenticated")
	}

	if savedSession.UserID() == nil {
		t.Fatal("expected saved session user ID")
	}

	if *savedSession.UserID() != newUser.ID {
		t.Fatalf(
			"expected saved session user ID %q, got %q",
			newUser.ID,
			*savedSession.UserID(),
		)
	}

	// Mora biti postavljen novi session cookie.
	setCookieHeaders := rec.Header().Values("Set-Cookie")

	if len(setCookieHeaders) == 0 {
		t.Fatal("expected Set-Cookie header")
	}

	var foundNewSessionCookie bool

	for _, header := range setCookieHeaders {
		if strings.HasPrefix(header, session.SessionCookieName()+"="+sess.ID()) {
			foundNewSessionCookie = true
			break
		}
	}

	if !foundNewSessionCookie {
		t.Fatalf(
			"expected Set-Cookie for session %q, got %v",
			sess.ID(),
			setCookieHeaders,
		)
	}
}
