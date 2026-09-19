package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/zeldojov/gopost/internal/session"
	"github.com/zeldojov/gopost/internal/user"
)

func TestRegister_InvalidUsername(t *testing.T) {
	st := setupTestStore(t)
	handler := NewHandler(st)

	sess := session.NewAnonSession(
		httptest.NewRequest(http.MethodGet, "/register", nil),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		strings.NewReader(url.Values{
			"username": {"ab"},
			"password": {"ValidPassword123!"},
		}.Encode()),
	)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusSeeOther,
			rec.Code,
		)
	}

	if location := rec.Header().Get("Location"); location != "/register" {
		t.Fatalf(
			"expected redirect to /register, got %q",
			location,
		)
	}

	if message, ok := sess.GetFlash("error"); !ok {
		t.Fatal("expected flash error")
	} else if message != "invalid username" {
		t.Fatalf(
			"expected flash error %q, got %q",
			"invalid username",
			message,
		)
	}
}

func TestRegister_InvalidPassword(t *testing.T) {
	st := setupTestStore(t)
	handler := NewHandler(st)

	sess := session.NewAnonSession(
		httptest.NewRequest(http.MethodGet, "/register", nil),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		strings.NewReader(url.Values{
			"username": {"testuser"},
			"password": {"123"},
		}.Encode()),
	)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusSeeOther,
			rec.Code,
		)
	}

	if location := rec.Header().Get("Location"); location != "/register" {
		t.Fatalf(
			"expected redirect to /register, got %q",
			location,
		)
	}

	if message, ok := sess.GetFlash("error"); !ok {
		t.Fatal("expected flash error")
	} else if message != "invalid password" {
		t.Fatalf(
			"expected flash error %q, got %q",
			"invalid password",
			message,
		)
	}
}

func TestRegister_UsernameTaken(t *testing.T) {
	st := setupTestStore(t)
	handler := NewHandler(st)

	existingUser := user.CreateUser("testuser", "existing-hash")

	if err := st.SaveUser(existingUser); err != nil {
		t.Fatal(err)
	}

	sess := session.NewAnonSession(
		httptest.NewRequest(http.MethodGet, "/register", nil),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		strings.NewReader(url.Values{
			"username": {"testuser"},
			"password": {"ValidPassword123!"},
		}.Encode()),
	)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusSeeOther,
			rec.Code,
		)
	}

	if location := rec.Header().Get("Location"); location != "/register" {
		t.Fatalf(
			"expected redirect to /register, got %q",
			location,
		)
	}

	if message, ok := sess.GetFlash("error"); !ok {
		t.Fatal("expected flash error")
	} else if message != "username already exists" {
		t.Fatalf(
			"expected flash error %q, got %q",
			"username already exists",
			message,
		)
	}
}

func TestRegister_Success(t *testing.T) {
	st := setupTestStore(t)
	handler := NewHandler(st)

	initialReq := httptest.NewRequest(http.MethodGet, "/register", nil)

	sess := session.NewAnonSession(initialReq)

	if err := st.SaveSession(sess); err != nil {
		t.Fatal(err)
	}

	oldSessionID := sess.ID()

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		strings.NewReader(url.Values{
			"username": {"testuser"},
			"password": {"ValidPassword123!"},
		}.Encode()),
	)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName(),
		Value: oldSessionID,
	})

	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.Register(rec, req)

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

	foundUser, err := st.GetUserByUsername("testuser")
	if err != nil {
		t.Fatalf("failed to get registered user: %v", err)
	}

	if foundUser.Username != "testuser" {
		t.Fatalf(
			"expected username %q, got %q",
			"testuser",
			foundUser.Username,
		)
	}

	if foundUser.PasswordHash == "ValidPassword123!" {
		t.Fatal("password was stored in plaintext")
	}

	if sess.ID() == oldSessionID {
		t.Fatal("expected session ID to change")
	}

	if !sess.IsAuthenticated() {
		t.Fatal("expected session to be authenticated")
	}

	if sess.UserID() == nil {
		t.Fatal("expected user ID in session")
	}

	if *sess.UserID() != foundUser.ID {
		t.Fatalf(
			"expected session user ID %q, got %q",
			foundUser.ID,
			*sess.UserID(),
		)
	}

	savedSession := &session.Session{}

	if err := st.GetSessionById(savedSession, sess.ID()); err != nil {
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

	if *savedSession.UserID() != foundUser.ID {
		t.Fatalf(
			"expected saved session user ID %q, got %q",
			foundUser.ID,
			*savedSession.UserID(),
		)
	}

	setCookieHeaders := rec.Header().Values("Set-Cookie")

	if len(setCookieHeaders) == 0 {
		t.Fatal("expected Set-Cookie header")
	}

	var foundNewSessionCookie bool

	for _, header := range setCookieHeaders {
		if strings.HasPrefix(
			header,
			session.SessionCookieName()+"="+sess.ID(),
		) {
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
