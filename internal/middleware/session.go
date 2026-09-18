package middleware

import (
	"errors"
	"log"
	"net/http"

	"github.com/zeldojov/gopost/internal/session"
)

func Session(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			sess *session.Session
			err  error
		)

		switch r.Method {
		case http.MethodGet:
			sess, err = session.LoadSession(r)

			switch {
			case errors.Is(err, session.ErrSessionCookieNotFound):
				sess, err = session.CreateSession(w, r)

			case errors.Is(err, session.ErrSessionNotFound):
				session.UnsetSessionCookie(w)
				sess, err = session.CreateSession(w, r)
			}

		case http.MethodPost:
			sess, err = session.LoadSession(r)

			if errors.Is(err, session.ErrSessionCookieNotFound) || errors.Is(err, session.ErrSessionNotFound) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, session.SetSession(sess, r))

		if err := sess.Save(); err != nil {
			log.Printf("failed to save session %q: %v", sess.ID(), err)
		}
	})
}
