package session

import (
	"errors"
	"net/http"
)

func GetSessionID(r *http.Request, config CookieConfig) (string, error) {
	cookie, err := r.Cookie(config.Name)
	if errors.Is(err, http.ErrNoCookie) {
		return "", ErrSessionCookieNotFound
	}

	if err != nil {
		return "", err
	}

	return cookie.Value, nil
}

func SetSessionCookie(w http.ResponseWriter, config CookieConfig, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     config.Name,
		Value:    sessionID,
		Path:     config.Path,
		HttpOnly: config.HTTPOnly,
		Secure:   config.Secure,
		SameSite: config.SameSite,
	})
}

func DeleteSessionCookie(w http.ResponseWriter, config CookieConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     config.Name,
		Value:    "",
		Path:     config.Path,
		MaxAge:   -1,
		HttpOnly: config.HTTPOnly,
		Secure:   config.Secure,
		SameSite: config.SameSite,
	})
}
