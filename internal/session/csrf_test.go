package session

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCSRFMiddleware_GET(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	store.Middleware(testCookieConfig(), nil)(
		store.CSRFMiddleware()(handler),
	).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestCSRFMiddleware_POSTWithoutToken(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("expected handler not to be called")
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()

	store.Middleware(testCookieConfig(), nil)(
		store.CSRFMiddleware()(handler),
	).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusForbidden,
			rec.Code,
		)
	}
}

func TestCSRFMiddleware_POSTWithInvalidToken(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("expected handler not to be called")
	})

	// First GET creates the session and CSRF token.
	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getRec := httptest.NewRecorder()

	store.Middleware(testCookieConfig(), nil)(
		store.CSRFMiddleware()(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		),
	).ServeHTTP(getRec, getReq)

	sessionCookie := getRec.Result().Cookies()[0]

	postReq := httptest.NewRequest(http.MethodPost, "/", nil)
	postReq.AddCookie(sessionCookie)

	postReq.Form = make(url.Values)
	postReq.Form.Set("csrf_token", "invalid-token")

	postRec := httptest.NewRecorder()

	store.Middleware(testCookieConfig(), nil)(
		store.CSRFMiddleware()(handler),
	).ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusForbidden,
			postRec.Code,
		)
	}
}

func TestCSRFMiddleware_POSTWithValidToken(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// First GET creates the session and CSRF token.
	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getRec := httptest.NewRecorder()

	store.Middleware(testCookieConfig(), nil)(
		store.CSRFMiddleware()(handler),
	).ServeHTTP(getRec, getReq)

	sessionCookie := getRec.Result().Cookies()[0]
	sessionID := sessionCookie.Value

	sess, err := store.GetSession(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	csrfToken, ok := sess.Get("csrf_token")
	if !ok {
		t.Fatal("expected CSRF token in session")
	}

	form := url.Values{}
	form.Set("csrf_token", csrfToken)

	postReq := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader(form.Encode()),
	)
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(sessionCookie)

	postRec := httptest.NewRecorder()

	store.Middleware(testCookieConfig(), nil)(
		store.CSRFMiddleware()(handler),
	).ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			postRec.Code,
		)
	}
}

func TestCSRFMiddleware_GETCreatesCSRFToken(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := GetSession(r)
		if !ok {
			t.Fatal("expected session to be available")
		}

		token, ok := sess.Get("csrf_token")
		if !ok {
			t.Fatal("expected CSRF token to be available")
		}

		if token == "" {
			t.Fatal("expected CSRF token to have a value")
		}

		w.WriteHeader(http.StatusOK)
	})

	store.Middleware(testCookieConfig(), nil)(
		store.CSRFMiddleware()(handler),
	).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}
}

func TestCSRFMiddleware_GETPreservesExistingCSRFToken(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	var firstToken string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := GetSession(r)
		if !ok {
			t.Fatal("expected session to be available")
		}

		firstToken = GetCSRFFromSession(sess)
	})

	store.Middleware(testCookieConfig(), nil)(
		store.CSRFMiddleware()(handler),
	).ServeHTTP(rec, req)

	sessionCookie := rec.Result().Cookies()[0]

	// Second GET with the same session.
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(sessionCookie)

	var secondToken string

	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := GetSession(r)
		if !ok {
			t.Fatal("expected session to be available")
		}

		secondToken = GetCSRFFromSession(sess)
	})

	rec = httptest.NewRecorder()

	store.Middleware(testCookieConfig(), nil)(
		store.CSRFMiddleware()(handler),
	).ServeHTTP(rec, req)

	if firstToken == "" {
		t.Fatal("expected first CSRF token")
	}

	if secondToken != firstToken {
		t.Fatalf(
			"expected CSRF token to be preserved, got %q instead of %q",
			secondToken,
			firstToken,
		)
	}
}

func TestSession_GetCSRFToken(t *testing.T) {
	sess := Session{
		Data: make(map[string]string),
	}

	token := GetCSRFFromSession(&sess)

	if token == "" {
		t.Fatal("expected CSRF token")
	}

	storedToken, ok := sess.Get("csrf_token")
	if !ok {
		t.Fatal("expected CSRF token to be stored in session")
	}

	if storedToken != token {
		t.Fatalf("expected stored token %q, got %q", token, storedToken)
	}

	secondToken := GetCSRFFromSession(&sess)

	if secondToken != token {
		t.Fatalf("expected existing token %q, got %q", token, secondToken)
	}
}
