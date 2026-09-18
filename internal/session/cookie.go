package session

import (
	"errors"
	"net/http"
)

const (
	sessionCookieName     = "session_id"
	sessionCookiePath     = "/"
	sessionCookieSecure   = false
	sessionCookieHTTPOnly = true
	sessionCookieSameSite = http.SameSiteLaxMode
)

func SessionCookieName() string {
	return sessionCookieName
}

func GetSessionCookie(r *http.Request) (*http.Cookie, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if errors.Is(err, http.ErrNoCookie) {
		return nil, ErrSessionCookieNotFound
	}
	if err != nil {
		return nil, err
	}

	return cookie, nil
}

func SetSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     sessionCookiePath,
		HttpOnly: sessionCookieHTTPOnly,
		Secure:   sessionCookieSecure,
		SameSite: sessionCookieSameSite,
	})
}

func UnsetSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     sessionCookiePath,
		MaxAge:   -1,
		HttpOnly: sessionCookieHTTPOnly,
		Secure:   sessionCookieSecure,
		SameSite: sessionCookieSameSite,
	})
}
