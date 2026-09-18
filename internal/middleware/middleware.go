package middleware

import (
	"net/http"

	"github.com/zeldojov/gopost/internal/store"
)

func PublicChain(st *store.Store, handler http.Handler) http.Handler {
	return AllowedMethod(
		Session(
			st,
			ValidateSession(
				st,
				CSRF(handler),
			),
		),
	)
}

func GuestChain(st *store.Store, handler http.Handler) http.Handler {
	return AllowedMethod(
		Session(
			st,
			ValidateSession(
				st,
				CSRF(
					Guest(handler),
				),
			),
		),
	)
}

func AuthChain(st *store.Store, handler http.Handler) http.Handler {
	return AllowedMethod(
		Session(
			st,
			ValidateSession(
				st,
				CSRF(
					Auth(handler),
				),
			),
		),
	)
}
