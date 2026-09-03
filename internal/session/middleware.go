package session

import (
	"context"
	"errors"
	"net/http"
)

type contextKey struct{}

func Middleware(store *Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionID, err := GetSessionID(r)

			if errors.Is(err, ErrSessionCookieNotFound) {
				sessionID = createSession(w, r, store)
			} else if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			sess, err := store.GetSession(sessionID)

			if errors.Is(err, ErrSessionNotFound) {
				DeleteSessionCookie(w)

				sessionID = createSession(w, r, store)

				sess, err = store.GetSession(sessionID)
			}

			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), contextKey{}, &sess)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)

			_ = store.UpdateSession(sessionID, sess)
		})
	}
}

func Get(r *http.Request) (*Session, bool) {
	session, ok := r.Context().Value(contextKey{}).(*Session)
	return session, ok
}

func createSession(w http.ResponseWriter, r *http.Request, store *Store) string {
	sessionID := store.AddSession(r)
	SetSessionCookie(w, sessionID)

	return sessionID
}
