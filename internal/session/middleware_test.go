package session

import (
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func testCookieConfig() CookieConfig {
	return CookieConfig{
		Name:     "session_id",
		Path:     "/",
		Secure:   true,
		HTTPOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func TestSessionMiddleware_CreatesSession(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)
	config := testCookieConfig()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	store.Middleware(config, log.Default())(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	cookies := rec.Result().Cookies()

	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	if cookies[0].Name != config.Name {
		t.Fatalf("expected cookie %q, got %q", config.Name, cookies[0].Name)
	}

	if cookies[0].Value == "" {
		t.Fatal("expected session ID cookie to have a value")
	}
}
func TestSessionMiddleware_UsesExistingSession(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)
	config := testCookieConfig()

	req := httptest.NewRequest("GET", "/", nil)
	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req.AddCookie(&http.Cookie{
		Name:  config.Name,
		Value: sessionID,
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := GetSession(r)
		if !ok {
			t.Fatal("expected session to be available")
		}

		if sess == nil {
			t.Fatal("expected session to not be nil")
		}

		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()

	store.Middleware(config, log.Default())(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if len(rec.Result().Cookies()) != 0 {
		t.Fatal("expected no new session cookie")
	}
}
func TestSessionMiddleware_RecreatesMissingSession(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)
	config := testCookieConfig()

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  config.Name,
		Value: "invalid-session-id",
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := GetSession(r)
		if !ok {
			t.Fatal("expected session to be available")
		}

		if sess == nil {
			t.Fatal("expected session to not be nil")
		}

		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()

	store.Middleware(config, log.Default())(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	cookies := rec.Result().Cookies()

	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	if cookies[0].MaxAge != -1 {
		t.Fatalf(
			"expected first cookie to delete old session, got MaxAge %d",
			cookies[0].MaxAge,
		)
	}

	if cookies[1].Name != config.Name {
		t.Fatalf(
			"expected second cookie %q, got %q",
			config.Name,
			cookies[1].Name,
		)
	}

	if cookies[1].Value == "" {
		t.Fatal("expected new session ID cookie to have a value")
	}
}

func TestSessionMiddleware_UpdatesSession(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)
	config := testCookieConfig()

	req := httptest.NewRequest("GET", "/", nil)
	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req.AddCookie(&http.Cookie{
		Name:  config.Name,
		Value: sessionID,
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := GetSession(r)
		if !ok {
			t.Fatal("expected session to be available")
		}

		sess.Set("user_id", "123")

		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()

	store.Middleware(config, log.Default())(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	sess, err := store.GetSession(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	value, ok := sess.Get("user_id")
	if !ok {
		t.Fatal("expected user_id to be saved")
	}

	if value != "123" {
		t.Fatalf("expected user_id %q, got %q", "123", value)
	}
}

func TestSessionMiddleware_ExpiredSession(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)
	config := testCookieConfig()

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sess, err := store.GetSession(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sess.ExpiresAt = time.Now().Add(-time.Hour)

	err = store.UpdateSession(sessionID, sess)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req.AddCookie(&http.Cookie{
		Name:  config.Name,
		Value: sessionID,
	})

	var handlerSession *Session

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerSession, _ = GetSession(r)
	})

	store.Middleware(config, log.Default())(handler).ServeHTTP(rec, req)

	if handlerSession == nil {
		t.Fatal("expected session in handler")
	}

	if handlerSession.ExpiresAt.Before(time.Now()) {
		t.Fatal("expected new session to be valid")
	}

	cookies := rec.Result().Cookies()

	var newSessionID string

	for _, cookie := range cookies {
		if cookie.Name == config.Name && cookie.MaxAge != -1 {
			newSessionID = cookie.Value
		}
	}

	if newSessionID == "" {
		t.Fatal("expected new session cookie")
	}

	if newSessionID == sessionID {
		t.Fatal("expected a new session ID")
	}
}

func TestSessionMiddleware_RefreshesSession(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)
	config := testCookieConfig()

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	sessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sess, err := store.GetSession(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sess.ExpiresAt = time.Now().Add(10 * time.Minute)

	err = store.UpdateSession(sessionID, sess)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	oldExpiresAt := sess.ExpiresAt
	oldCreatedAt := sess.CreatedAt

	req.AddCookie(&http.Cookie{
		Name:  config.Name,
		Value: sessionID,
	})

	var handlerSession *Session

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerSession, _ = GetSession(r)
	})

	store.Middleware(config, log.Default())(handler).ServeHTTP(rec, req)

	if handlerSession == nil {
		t.Fatal("expected session in handler")
	}

	if !handlerSession.ExpiresAt.After(oldExpiresAt) {
		t.Fatal("expected session expiration to be extended")
	}

	if !handlerSession.CreatedAt.Equal(oldCreatedAt) {
		t.Fatal("expected existing session to be preserved")
	}
}

func TestSessionMiddleware_RegeneratesSessionID(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)
	config := testCookieConfig()

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	oldSessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sess, err := store.GetSession(oldSessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sess.CreatedAt = time.Now().Add(-8 * 24 * time.Hour)

	if err := store.UpdateSession(oldSessionID, sess); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req.AddCookie(&http.Cookie{
		Name:  config.Name,
		Value: oldSessionID,
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	store.Middleware(config, log.Default())(handler).ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()

	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	newSessionID := cookies[0].Value

	if newSessionID == oldSessionID {
		t.Fatal("expected session ID to be regenerated")
	}
}

func TestSessionMiddleware_RegeneratesSessionID_PreservesSession(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)
	config := testCookieConfig()

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	oldSessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sess, err := store.GetSession(oldSessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sess.CreatedAt = time.Now().Add(-8 * 24 * time.Hour)
	sess.Set("user_id", "123")

	if err := store.UpdateSession(oldSessionID, sess); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req.AddCookie(&http.Cookie{
		Name:  config.Name,
		Value: oldSessionID,
	})

	var handlerSession *Session

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerSession, _ = GetSession(r)
	})

	store.Middleware(config, log.Default())(handler).ServeHTTP(rec, req)

	if handlerSession == nil {
		t.Fatal("expected session in handler")
	}

	if value, ok := handlerSession.Get("user_id"); !ok || value != "123" {
		t.Fatal("expected session data to be preserved")
	}

	if _, err := store.GetSession(oldSessionID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected old session ID to be removed, got %v", err)
	}
}

func TestSessionMiddleware_RegeneratedSessionIDIsValid(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)
	config := testCookieConfig()

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	oldSessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sess, err := store.GetSession(oldSessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sess.CreatedAt = time.Now().Add(-8 * 24 * time.Hour)

	if err := store.UpdateSession(oldSessionID, sess); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req.AddCookie(&http.Cookie{
		Name:  config.Name,
		Value: oldSessionID,
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	store.Middleware(config, log.Default())(handler).ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()

	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	newSessionID := cookies[0].Value

	if newSessionID == oldSessionID {
		t.Fatal("expected session ID to be regenerated")
	}

	_, err = store.GetSession(newSessionID)
	if err != nil {
		t.Fatalf("expected regenerated session to be valid: %v", err)
	}
}

func TestSessionMiddleware_RecreatesSessionOnIPMismatch(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)
	config := testCookieConfig()

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	req.Header.Set("User-Agent", "test-agent")

	oldSessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req.AddCookie(&http.Cookie{
		Name:  config.Name,
		Value: oldSessionID,
	})

	// Simulate request coming from a different IP.
	req.RemoteAddr = "192.168.1.100:8080"

	var handlerSession *Session

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerSession, _ = GetSession(r)
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()

	store.Middleware(config, log.Default())(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if handlerSession == nil {
		t.Fatal("expected session in handler")
	}

	if handlerSession.IP != "192.168.1.100" {
		t.Fatalf(
			"expected new session IP %q, got %q",
			"192.168.1.100",
			handlerSession.IP,
		)
	}

	if _, err := store.GetSession(oldSessionID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected old session to be removed, got %v", err)
	}

	cookies := rec.Result().Cookies()

	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	if cookies[0].MaxAge != -1 {
		t.Fatalf(
			"expected first cookie to delete old session, got MaxAge %d",
			cookies[0].MaxAge,
		)
	}

	newSessionID := cookies[1].Value

	if newSessionID == "" {
		t.Fatal("expected new session ID")
	}

	if newSessionID == oldSessionID {
		t.Fatal("expected new session ID to differ from old session ID")
	}
}

func TestSessionMiddleware_RecreatesSessionOnUserAgentMismatch(t *testing.T) {
	repository := &mockSessionRepository{}
	store := NewStore(repository)
	config := testCookieConfig()

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	req.Header.Set("User-Agent", "test-agent")

	oldSessionID, err := store.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req.AddCookie(&http.Cookie{
		Name:  config.Name,
		Value: oldSessionID,
	})

	// Simulate request with a different User-Agent.
	req.Header.Set("User-Agent", "different-agent")

	var handlerSession *Session

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerSession, _ = GetSession(r)
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()

	store.Middleware(config, log.Default())(handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if handlerSession == nil {
		t.Fatal("expected session in handler")
	}

	if handlerSession.UserAgent != "different-agent" {
		t.Fatalf(
			"expected new session User-Agent %q, got %q",
			"different-agent",
			handlerSession.UserAgent,
		)
	}

	if _, err := store.GetSession(oldSessionID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected old session to be removed, got %v", err)
	}

	cookies := rec.Result().Cookies()

	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	if cookies[0].MaxAge != -1 {
		t.Fatalf(
			"expected first cookie to delete old session, got MaxAge %d",
			cookies[0].MaxAge,
		)
	}

	newSessionID := cookies[1].Value

	if newSessionID == "" {
		t.Fatal("expected new session ID")
	}

	if newSessionID == oldSessionID {
		t.Fatal("expected new session ID to differ from old session ID")
	}
}

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

		firstToken = sess.GetCSRFToken()
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

		secondToken = sess.GetCSRFToken()
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
