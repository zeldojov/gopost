package middleware

import (
	"log"
	"net/http"

	"github.com/zeldojov/gopost/internal/session"
)

func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			sess, ok := session.GetSession(r)
			if !ok {
				log.Printf("session missing from request")
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			if !sess.ValidateCSRFToken(r) {
				log.Println("csrf token failed validation")
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
