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
)

func loadTestSession(t *testing.T, id string) *session.Session {
	t.Helper()

	sess := &session.Session{}

	if err := sess.Load(id); err != nil {
		t.Fatal(err)
	}

	return sess
}

func publicChain(handler http.Handler) http.Handler {
	return middleware.AllowedMethod(
		middleware.Session(
			middleware.ValidateSession(
				middleware.CSRF(handler),
			),
		),
	)
}

func TestLoginPage(t *testing.T) {
	setupTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)

	sess := session.NewAnonSession(req)

	if err := sess.Save(); err != nil {
		t.Fatal(err)
	}

	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	LoginPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	body := rec.Body.String()

	if !strings.Contains(body, `action="/login"`) {
		t.Fatal(`expected form action "/login"`)
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
func TestLoginPage_MissingSession(t *testing.T) {
	setupTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)

	rec := httptest.NewRecorder()

	LoginPage(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			rec.Code,
		)
	}
}

func TestLoginPage_ShowsFlashError(t *testing.T) {
	setupTestDB(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/login",
		nil,
	)

	sess := session.NewAnonSession(req)
	sess.SetFlash("error", "invalid credentials")

	ctx := context.WithValue(req.Context(), session.ContextKey{}, sess)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	LoginPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	body := rec.Body.String()

	if !strings.Contains(body, "invalid credentials") {
		t.Fatal(`expected body to contain "invalid credentials"`)
	}

	if sess.HasValue("flash:error") {
		t.Fatal("expected flash error to be consumed")
	}
}

func TestLoginFlashFlow(t *testing.T) {
	setupTestDB(t)

	loginHandler := publicChain(
		http.HandlerFunc(Login),
	)

	loginPageHandler := publicChain(
		http.HandlerFunc(LoginPage),
	)

	// GET /login — kreira anonimnu sesiju.
	getReq := httptest.NewRequest(
		http.MethodGet,
		"/login",
		nil,
	)

	getRec := httptest.NewRecorder()

	loginPageHandler.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf(
			"expected GET /login status %d, got %d",
			http.StatusOK,
			getRec.Code,
		)
	}

	cookie := parseSessionCookie(t, getRec)

	sess := loadTestSession(t, cookie.Value)

	// POST /login sa pogrešnim credentials.
	form := url.Values{
		"username":   {"nonexistent"},
		"password":   {"WrongPassword123!"},
		"csrf_token": {sess.CSRFToken()},
	}

	postReq := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(form.Encode()),
	)

	postReq.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	postReq.AddCookie(cookie)

	postRec := httptest.NewRecorder()

	loginHandler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusSeeOther {
		t.Fatalf(
			"expected POST /login status %d, got %d",
			http.StatusSeeOther,
			postRec.Code,
		)
	}

	if location := postRec.Header().Get("Location"); location != "/login" {
		t.Fatalf(
			"expected redirect to /login, got %q",
			location,
		)
	}

	// Flash mora biti sačuvan u DB.
	savedSession := loadTestSession(t, cookie.Value)

	if !savedSession.HasValue("flash:error") {
		t.Fatal("expected flash error to be saved")
	}

	// GET /login — isti session, flash treba da se prikaže.
	finalReq := httptest.NewRequest(
		http.MethodGet,
		"/login",
		nil,
	)

	finalReq.AddCookie(cookie)

	finalRec := httptest.NewRecorder()

	loginPageHandler.ServeHTTP(finalRec, finalReq)

	if finalRec.Code != http.StatusOK {
		t.Fatalf(
			"expected GET /login status %d, got %d",
			http.StatusOK,
			finalRec.Code,
		)
	}

	if !strings.Contains(
		finalRec.Body.String(),
		"invalid credentials",
	) {
		t.Fatal(`expected page to contain "invalid credentials"`)
	}

	// Flash mora biti potrošen i sačuvan kao obrisan.
	savedSession = loadTestSession(t, cookie.Value)

	if savedSession.HasValue("flash:error") {
		t.Fatal("expected flash error to be consumed")
	}
}
