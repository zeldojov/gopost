package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/zeldojov/gopost/internal/middleware"
	"github.com/zeldojov/gopost/internal/session"
	"github.com/zeldojov/gopost/internal/store"
)

func parseSessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	resp := rec.Result()

	for _, cookie := range resp.Cookies() {
		if cookie.Name == session.SessionCookieName() {
			return cookie
		}
	}

	t.Fatal("session cookie not found")
	return nil
}

func publicChain(st *store.Store, handler http.Handler) http.Handler {
	return middleware.AllowedMethod(
		middleware.Session(
			st,
			middleware.ValidateSession(
				st,
				middleware.CSRF(handler),
			),
		),
	)
}

func TestRegisterPage(t *testing.T) {
	st := setupTestStore(t)
	handler := NewHandler(st)

	req := httptest.NewRequest(
		http.MethodGet,
		"/register",
		nil,
	)

	sess := session.NewAnonSession(req)

	ctx := context.WithValue(
		req.Context(),
		session.ContextKey{},
		sess,
	)

	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.RegisterPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	body := rec.Body.String()

	if !strings.Contains(body, `action="/register"`) {
		t.Fatal(`expected form action "/register"`)
	}

	if !strings.Contains(body, `method="POST"`) {
		t.Fatal(`expected form method "POST"`)
	}

	expectedCSRF := `<input type="hidden" name="csrf_token" value="` +
		sess.CSRFToken() +
		`">`

	if !strings.Contains(body, expectedCSRF) {
		t.Fatal("expected session CSRF token in form")
	}
}

func TestRegisterPage_MissingSession(t *testing.T) {
	st := setupTestStore(t)
	handler := NewHandler(st)

	req := httptest.NewRequest(
		http.MethodGet,
		"/register",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.RegisterPage(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}
}

func TestRegisterPage_ShowsFlashError(t *testing.T) {
	st := setupTestStore(t)
	handler := NewHandler(st)

	req := httptest.NewRequest(
		http.MethodGet,
		"/register",
		nil,
	)

	sess := session.NewAnonSession(req)
	sess.SetFlash("error", "invalid username")

	ctx := context.WithValue(
		req.Context(),
		session.ContextKey{},
		sess,
	)

	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	handler.RegisterPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	body := rec.Body.String()

	if !strings.Contains(body, "invalid username") {
		t.Fatal(`expected body to contain "invalid username"`)
	}

	if sess.HasValue("flash:error") {
		t.Fatal("expected flash error to be consumed")
	}
}

func TestRegisterFlashFlow(t *testing.T) {
	st := setupTestStore(t)
	handler := NewHandler(st)

	registerHandler := publicChain(
		st,
		http.HandlerFunc(handler.Register),
	)

	registerPageHandler := publicChain(
		st,
		http.HandlerFunc(handler.RegisterPage),
	)

	// GET /register — kreira anonimnu sesiju.
	getReq := httptest.NewRequest(
		http.MethodGet,
		"/register",
		nil,
	)

	getRec := httptest.NewRecorder()

	registerPageHandler.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf(
			"expected GET /register status %d, got %d",
			http.StatusOK,
			getRec.Code,
		)
	}

	cookie := parseSessionCookie(t, getRec)

	sess := loadTestSession(t, st, cookie.Value)

	// POST /register sa nevalidnim username-om.
	form := url.Values{
		"username":   {"ab"},
		"password":   {"ValidPassword123!"},
		"csrf_token": {sess.CSRFToken()},
	}

	postReq := httptest.NewRequest(
		http.MethodPost,
		"/register",
		strings.NewReader(form.Encode()),
	)

	postReq.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	postReq.AddCookie(cookie)

	postRec := httptest.NewRecorder()

	registerHandler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusSeeOther {
		t.Fatalf(
			"expected POST /register status %d, got %d",
			http.StatusSeeOther,
			postRec.Code,
		)
	}

	if location := postRec.Header().Get("Location"); location != "/register" {
		t.Fatalf(
			"expected redirect to /register, got %q",
			location,
		)
	}

	// Flash mora biti sačuvan u DB.
	savedSession := loadTestSession(t, st, cookie.Value)

	if !savedSession.HasValue("flash:error") {
		t.Fatal("expected flash error to be saved")
	}

	// GET /register — flash treba da se prikaže.
	finalReq := httptest.NewRequest(
		http.MethodGet,
		"/register",
		nil,
	)

	finalReq.AddCookie(cookie)

	finalRec := httptest.NewRecorder()

	registerPageHandler.ServeHTTP(finalRec, finalReq)

	if finalRec.Code != http.StatusOK {
		t.Fatalf(
			"expected GET /register status %d, got %d",
			http.StatusOK,
			finalRec.Code,
		)
	}

	if !strings.Contains(
		finalRec.Body.String(),
		"invalid username",
	) {
		t.Fatal(`expected page to contain "invalid username"`)
	}

	// Flash mora biti potrošen.
	savedSession = loadTestSession(t, st, cookie.Value)

	if savedSession.HasValue("flash:error") {
		t.Fatal("expected flash error to be consumed")
	}
}
