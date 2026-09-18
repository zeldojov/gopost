package middleware

import (
	"net/http"

	"github.com/zeldojov/gopost/internal/session"
)

func ValidateSession(next http.Handler) http.Handler {
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
				if err := sess.Recreate(w, r); err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
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
