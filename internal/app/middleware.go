package app

import (
	"net/http"
)

type Middleware func(http.Handler) http.Handler

func Chain(middlewares ...Middleware) Middleware {
	if len(middlewares) == 0 {
		panic("middleware chain cannot be empty")
	}

	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}

		return next
	}
}

func (a *App) SessionMiddleware(next http.Handler) http.Handler {
	return a.sessionStore.Middleware(
		a.config.Session,
		a.logger,
	)(next)
}

func (a *App) CSRFMiddleware(next http.Handler) http.Handler {
	return a.sessionStore.CSRFMiddleware()(next)
}

func (a *App) MethodMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			http.Error(
				w,
				http.StatusText(http.StatusMethodNotAllowed),
				http.StatusMethodNotAllowed,
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}
