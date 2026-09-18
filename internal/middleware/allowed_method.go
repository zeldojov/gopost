package middleware

import "net/http"

func AllowedMethod(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodPost:
			next.ServeHTTP(w, r)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
