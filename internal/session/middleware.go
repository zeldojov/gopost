package session

import (
	"context"
	"errors"
	"log"
	"net/http"
)

func (s *Store) Middleware(config CookieConfig, logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			sessionID, err := GetSessionID(r)

			if errors.Is(err, ErrSessionCookieNotFound) {
				sessionID, err = s.createSession(w, r, config)
				if err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
			} else if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			sess, err := s.GetSession(sessionID)

			if errors.Is(err, ErrSessionNotFound) ||
				errors.Is(err, ErrSessionExpired) {

				DeleteSessionCookie(w, config)

				sessionID, err = s.createSession(w, r, config)
				if err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}

				sess, err = s.GetSession(sessionID)
			}

			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			matches, err := s.MatchesRequest(sessionID, r)
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			if !matches {
				if err := s.RemoveSession(sessionID); err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}

				DeleteSessionCookie(w, config)

				sessionID, err = s.createSession(w, r, config)
				if err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}

				sess, err = s.GetSession(sessionID)
				if err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
			}

			needsRegeneration, err := s.NeedsRegeneration(sessionID)
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			if needsRegeneration {
				sessionID, err = s.RegenerateSessionID(sessionID)
				if err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}

				SetSessionCookie(w, config, sessionID)

				sess, err = s.GetSession(sessionID)
				if err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
			}

			sess, err = s.RefreshSession(sessionID)
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), contextKey{}, &sess)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)

			if err := s.UpdateSession(sessionID, sess); err != nil {
				logger.Printf(
					"failed to update session %q: %v",
					sessionID,
					err,
				)
			}
		})
	}
}

func (s *Store) CSRFMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, ok := GetSession(r)
			if !ok {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			if r.Method == http.MethodGet {
				sess.GetCSRFToken()
				next.ServeHTTP(w, r)
				return
			}

			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			expectedToken, ok := sess.Get("csrf_token")
			if !ok || expectedToken == "" {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			requestToken := r.FormValue("csrf_token")

			if requestToken == "" || requestToken != expectedToken {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
