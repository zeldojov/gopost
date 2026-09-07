package app

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/zeldojov/gopost/internal/session"
)

// region session
func TestChain(t *testing.T) {
	var calls []string

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "middleware1-before")
			next.ServeHTTP(w, r)
			calls = append(calls, "middleware1-after")
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "middleware2-before")
			next.ServeHTTP(w, r)
			calls = append(calls, "middleware2-after")
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "handler")
	})

	chain := Chain(middleware1, middleware2)
	chain(handler).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	expected := []string{
		"middleware1-before",
		"middleware2-before",
		"handler",
		"middleware2-after",
		"middleware1-after",
	}

	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("expected calls %v, got %v", expected, calls)
	}
}

func TestChain_CanBeReused(t *testing.T) {
	var calls []string

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, "middleware")
			next.ServeHTTP(w, r)
		})
	}

	chain := Chain(middleware)

	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "handler1")
	})

	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "handler2")
	})

	chain(handler1).ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/", nil),
	)

	chain(handler2).ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/", nil),
	)

	expected := []string{
		"middleware",
		"handler1",
		"middleware",
		"handler2",
	}

	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("expected calls %v, got %v", expected, calls)
	}
}

func TestChain_SingleMiddleware(t *testing.T) {
	called := false

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	Chain(middleware)(
		handler,
	).ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest("GET", "/", nil),
	)

	if !called {
		t.Fatal("expected middleware to be called")
	}
}

func TestChain_Empty(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected Chain to panic with no middleware")
		}
	}()

	Chain()
}

func TestApp_SessionMiddleware_CreatesSession(t *testing.T) {
	app, err := New(Config{
		DBPath: ":memory:",
		Session: session.CookieConfig{
			Name:     "session_id",
			Path:     "/",
			HTTPOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer app.Close()

	handler := app.SessionMiddleware(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {},
	))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()

	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	if cookies[0].Name != "session_id" {
		t.Fatalf("expected session_id cookie, got %q", cookies[0].Name)
	}

	if cookies[0].Value == "" {
		t.Fatal("expected session cookie value")
	}
}

func TestApp_SessionMiddleware_UsesExistingSession(t *testing.T) {
	app, err := New(Config{
		DBPath: ":memory:",
		Session: session.CookieConfig{
			Name:     "session_id",
			Path:     "/",
			HTTPOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer app.Close()

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	sessionID, err := app.sessionStore.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID,
	})

	handler := app.SessionMiddleware(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {},
	))

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()

	if len(cookies) != 0 {
		t.Fatalf("expected no new cookie, got %d", len(cookies))
	}
}

func TestApp_SessionMiddleware_RecreatesMissingSession(t *testing.T) {
	app, err := New(Config{
		DBPath: ":memory:",
		Session: session.CookieConfig{
			Name:     "session_id",
			Path:     "/",
			HTTPOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer app.Close()

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: "missing-session",
	})

	handler := app.SessionMiddleware(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {},
	))

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()

	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	if cookies[0].Name != "session_id" || cookies[0].MaxAge != -1 {
		t.Fatal("expected old session cookie to be deleted")
	}

	if cookies[1].Name != "session_id" || cookies[1].Value == "" {
		t.Fatal("expected new session cookie")
	}

	if cookies[1].Value == "missing-session" {
		t.Fatal("expected a new session ID")
	}
}
func TestApp_SessionMiddleware_RecreatesExpiredSession(t *testing.T) {
	app, err := New(Config{
		DBPath: ":memory:",
		Session: session.CookieConfig{
			Name:     "session_id",
			Path:     "/",
			HTTPOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer app.Close()

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	sessionID, err := app.sessionStore.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = app.db.Exec(`
		UPDATE sessions
		SET expires_at = ?
		WHERE id = ?
	`, time.Now().Add(-time.Minute), sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID,
	})

	handler := app.SessionMiddleware(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {},
	))

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()

	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	if cookies[0].Name != "session_id" || cookies[0].MaxAge != -1 {
		t.Fatal("expected old session cookie to be deleted")
	}

	if cookies[1].Name != "session_id" || cookies[1].Value == "" {
		t.Fatal("expected new session cookie")
	}

	if cookies[1].Value == sessionID {
		t.Fatal("expected a new session ID")
	}
}

func TestApp_SessionMiddleware_RecreatesSessionOnIPMismatch(t *testing.T) {
	app, err := New(Config{
		DBPath: ":memory:",
		Session: session.CookieConfig{
			Name:     "session_id",
			Path:     "/",
			HTTPOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer app.Close()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	sessionID, err := app.sessionStore.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req.RemoteAddr = "192.168.1.10:1234"

	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID,
	})

	handler := app.SessionMiddleware(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {},
	))

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()

	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	if cookies[0].Name != "session_id" || cookies[0].MaxAge != -1 {
		t.Fatal("expected old session cookie to be deleted")
	}

	if cookies[1].Name != "session_id" || cookies[1].Value == "" {
		t.Fatal("expected new session cookie")
	}

	if cookies[1].Value == sessionID {
		t.Fatal("expected a new session ID")
	}
}
func TestApp_SessionMiddleware_RecreatesSessionOnUserAgentMismatch(t *testing.T) {
	app, err := New(Config{
		DBPath: ":memory:",
		Session: session.CookieConfig{
			Name:     "session_id",
			Path:     "/",
			HTTPOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer app.Close()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("User-Agent", "Browser-A")

	sessionID, err := app.sessionStore.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req.Header.Set("User-Agent", "Browser-B")

	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID,
	})

	handler := app.SessionMiddleware(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {},
	))

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()

	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	if cookies[0].Name != "session_id" || cookies[0].MaxAge != -1 {
		t.Fatal("expected old session cookie to be deleted")
	}

	if cookies[1].Name != "session_id" || cookies[1].Value == "" {
		t.Fatal("expected new session cookie")
	}

	if cookies[1].Value == sessionID {
		t.Fatal("expected a new session ID")
	}
}

// endregion session
// region csrf
func TestApp_CSRFMiddleware_CreatesToken(t *testing.T) {
	app, err := New(Config{
		DBPath: ":memory:",
		Session: session.CookieConfig{
			Name:     "session_id",
			Path:     "/",
			HTTPOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer app.Close()

	called := false

	handler := app.SessionMiddleware(
		app.CSRFMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true

				sess, ok := session.GetSession(r)
				if !ok {
					t.Fatal("expected session")
				}

				if sess.GetCSRFToken() == "" {
					t.Fatal("expected CSRF token")
				}
			}),
		),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("expected handler to be called")
	}
}
func TestApp_CSRFMiddleware_RejectsPOSTWithoutToken(t *testing.T) {
	app, err := New(Config{
		DBPath: ":memory:",
		Session: session.CookieConfig{
			Name:     "session_id",
			Path:     "/",
			HTTPOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer app.Close()

	handler := app.SessionMiddleware(
		app.CSRFMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					t.Fatal("expected POST handler not to be called")
				}
			}),
		),
	)

	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getRec := httptest.NewRecorder()

	handler.ServeHTTP(getRec, getReq)

	cookies := getRec.Result().Cookies()

	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	postReq := httptest.NewRequest(http.MethodPost, "/", nil)
	postReq.AddCookie(cookies[0])

	postRec := httptest.NewRecorder()

	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusForbidden,
			postRec.Code,
		)
	}
}
func TestApp_CSRFMiddleware_RejectsPOSTWithInvalidToken(t *testing.T) {
	app, err := New(Config{
		DBPath: ":memory:",
		Session: session.CookieConfig{
			Name:     "session_id",
			Path:     "/",
			HTTPOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer app.Close()

	handler := app.SessionMiddleware(
		app.CSRFMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					t.Fatal("expected POST handler not to be called")
				}
			}),
		),
	)

	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getRec := httptest.NewRecorder()

	handler.ServeHTTP(getRec, getReq)

	cookies := getRec.Result().Cookies()

	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	postReq := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader("csrf_token=invalid-token"),
	)
	postReq.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)
	postReq.AddCookie(cookies[0])

	postRec := httptest.NewRecorder()

	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusForbidden,
			postRec.Code,
		)
	}
}
func TestApp_CSRFMiddleware_AllowsPOSTWithValidToken(t *testing.T) {
	app, err := New(Config{
		DBPath: ":memory:",
		Session: session.CookieConfig{
			Name:     "session_id",
			Path:     "/",
			HTTPOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer app.Close()

	called := false

	handler := app.SessionMiddleware(
		app.CSRFMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
			}),
		),
	)

	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getRec := httptest.NewRecorder()

	handler.ServeHTTP(getRec, getReq)

	cookies := getRec.Result().Cookies()

	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	sessionID := cookies[0].Value

	sess, err := app.sessionStore.GetSession(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	csrfToken := sess.GetCSRFToken()

	if csrfToken == "" {
		t.Fatal("expected CSRF token")
	}

	postReq := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader("csrf_token="+url.QueryEscape(csrfToken)),
	)
	postReq.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)
	postReq.AddCookie(cookies[0])

	postRec := httptest.NewRecorder()

	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			postRec.Code,
		)
	}

	if !called {
		t.Fatal("expected POST handler to be called")
	}
}

// endregion csrf
// region method
func TestApp_MethodMiddleware(t *testing.T) {
	app := &App{}

	handler := app.MethodMiddleware(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	))

	tests := []struct {
		method string
		status int
	}{
		{http.MethodGet, http.StatusNoContent},
		{http.MethodPost, http.StatusNoContent},
		{http.MethodPut, http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Fatalf(
					"expected status %d, got %d",
					tt.status,
					rec.Code,
				)
			}
		})
	}
}

func TestApp_MiddlewareChain_RejectsMethodBeforeSession(t *testing.T) {
	app, err := New(Config{
		DBPath: ":memory:",
		Session: session.CookieConfig{
			Name:     "session_id",
			Path:     "/",
			HTTPOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer app.Close()

	handler := Chain(
		app.MethodMiddleware,
		app.SessionMiddleware,
		app.CSRFMiddleware,
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("expected handler not to be called")
	}))

	req := httptest.NewRequest(http.MethodPut, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusMethodNotAllowed,
			rec.Code,
		)
	}

	var count int

	err = app.db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&count)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 0 {
		t.Fatalf("expected no session to be created, got %d", count)
	}
}

// endregion method
