package middleware

import (
	"net/http"

	"github.com/zeldojov/gopost/internal/session"
	"github.com/zeldojov/gopost/internal/store"
)

func ValidateSession(st *store.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := session.GetSession(r)
		if !ok {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if sess.IsExpired() ||
			!sess.MatchUserAgent(r) ||
			!sess.MatchIP(r) {

			if r.Method == http.MethodGet {
				if err := st.DeleteSession(sess); err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}

				session.UnsetSessionCookie(w)

				sess.Recreate(w, r)

				if err := st.SaveSession(sess); err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}

				session.SetSessionCookie(w, sess.ID())
			} else {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}

		if sess.ShouldRefresh() {
			sess.Touch()
		}

		next.ServeHTTP(w, r)
	})
}
