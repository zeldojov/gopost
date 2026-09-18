package middleware

import (
	"log"
	"net/http"

	"github.com/zeldojov/gopost/internal/session"
)

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := session.GetSession(r)
		if !ok {
			log.Printf("session missing from request")
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if !sess.IsAuthenticated() {
			if r.Method == http.MethodGet {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
