package session

import (
	"errors"
	"net/http"
)

var ErrSessionCookieNotFound = errors.New("session cookie not found")

func GetSessionID(r *http.Request) (string, error) {
	cookie, err := r.Cookie("session_id")
	if errors.Is(err, http.ErrNoCookie) {
		return "", ErrSessionCookieNotFound
	}

	if err != nil {
		return "", err
	}

	return cookie.Value, nil
}

func SetSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
	})
}

func DeleteSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}
