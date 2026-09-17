package main

import (
	"context"
	"errors"
	"net/http"
)

func AllowedMethodMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodPost:
			next.ServeHTTP(w, r)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			sess *Session
			err  error
		)

		switch r.Method {
		case http.MethodGet:
			sess, err = loadSession(r)

			switch {
			case errors.Is(err, ErrSessionCookieNotFound):
				sess, err = createSession(w, r)

			case errors.Is(err, ErrSessionNotFound):
				unsetSessionCookie(w)
				sess, err = createSession(w, r)
			}

		case http.MethodPost:
			sess, err = loadSession(r)

			if errors.Is(err, ErrSessionCookieNotFound) ||
				errors.Is(err, ErrSessionNotFound) {
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

		ctx := context.WithValue(r.Context(), contextKey{}, sess)
		next.ServeHTTP(w, r.WithContext(ctx))

		if err := sess.Save(); err != nil {
			LOG.Printf("failed to save session %q: %v", sess.id, err)
		}
	})
}

func ValidateSessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := GetSession(r)
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

func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			sess, ok := GetSession(r)
			if !ok {
				LOG.Printf("session missing from request")
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			if !sess.ValidateCSRFToken(r) {
				LOG.Println("csrf token failed validation")
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := GetSession(r)
		if !ok {
			LOG.Printf("session missing from request")
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if !sess.IsAuthenticated() {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
