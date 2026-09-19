package middleware

import (
	"errors"
	"log"
	"net/http"

	"github.com/zeldojov/gopost/internal/session"
	"github.com/zeldojov/gopost/internal/store"
)

func Session(st *store.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sess *session.Session
		newSession := false

		cookie, err := session.GetSessionCookie(r)

		switch {
		case errors.Is(err, session.ErrSessionCookieNotFound):
			if r.Method == http.MethodPost {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			sess = session.NewAnonSession(r)
			newSession = true

		case err != nil:
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return

		default:
			sess = &session.Session{}

			if err := st.GetSessionById(sess, cookie.Value); err != nil {
				if errors.Is(err, session.ErrSessionNotFound) {
					if r.Method == http.MethodPost {
						http.Error(w, "forbidden", http.StatusForbidden)
						return
					}

					session.UnsetSessionCookie(w)

					sess = session.NewAnonSession(r)
					newSession = true
				} else {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
			}
		}

		if newSession {
			if err := st.SaveSession(sess); err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			session.SetSessionCookie(w, sess.ID())
		}

		r = session.SetSession(sess, r)

		next.ServeHTTP(w, r)

		if err := st.SaveSession(sess); err != nil {
			log.Printf(
				"failed to save session %q: %v",
				sess.ID(),
				err,
			)
		}
	})
}
