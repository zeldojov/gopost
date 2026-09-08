package session

import "net/http"

func (s *Store) CSRFMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, ok := GetSession(r)
			if !ok {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			if r.Method == http.MethodGet {
				GetCSRFFromSession(sess)
				next.ServeHTTP(w, r)
				return
			}

			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			if !ValidateCSRFToken(sess, r) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func CreateCSRFToken(session *Session) string {
	token := newRandomToken()
	session.Set("csrf_token", token)

	return token
}

func GetCSRFFromSession(session *Session) string {
	if token, ok := session.Get("csrf_token"); ok {
		return token
	}

	return CreateCSRFToken(session)
}

func ValidateCSRFToken(session *Session, r *http.Request) bool {
	expectedToken, ok := session.Get("csrf_token")
	if !ok || expectedToken == "" {
		return false
	}

	requestToken := r.FormValue("csrf_token")

	return requestToken != "" && requestToken == expectedToken
}
