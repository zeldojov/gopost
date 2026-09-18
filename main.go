package main

import (
	"log"
	"net/http"

	"github.com/zeldojov/gopost/internal/handlers"
	"github.com/zeldojov/gopost/internal/middleware"
	"github.com/zeldojov/gopost/internal/store"
)

var LOG = log.Default()

var err error

func main() {
	store, err := InitDB()
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	mux := http.DefaultServeMux

	h := handlers.NewHandler(store)

	http.Handle("/login", guestChain(store, http.HandlerFunc(h.Login)))
	http.Handle("/login/", guestChain(store, http.HandlerFunc(h.LoginPage)))

	http.Handle("/register", guestChain(store, http.HandlerFunc(h.Register)))
	http.Handle("/register/", guestChain(store, http.HandlerFunc(h.RegisterPage)))

	http.Handle("/user/home", authChain(store, http.HandlerFunc(h.UserHome)))
	http.Handle("/logout", authChain(store, http.HandlerFunc(h.Logout)))

	handler := middleware.AllowedMethod(
		middleware.Session(
			store,
			middleware.ValidateSession(
				store,
				middleware.CSRF(mux),
			),
		),
	)

	if err := http.ListenAndServe("127.0.0.1:8000", handler); err != nil {
		LOG.Fatal(err)
	}

}

func publicChain(st *store.Store, handler http.Handler) http.Handler {
	return middleware.AllowedMethod(
		middleware.Session(
			st,
			middleware.ValidateSession(
				st,
				middleware.CSRF(handler),
			),
		),
	)
}

func guestChain(st *store.Store, handler http.Handler) http.Handler {
	return middleware.AllowedMethod(
		middleware.Session(
			st,
			middleware.ValidateSession(
				st,
				middleware.CSRF(
					middleware.Guest(handler),
				),
			),
		),
	)
}

func authChain(st *store.Store, handler http.Handler) http.Handler {
	return middleware.AllowedMethod(
		middleware.Session(
			st,
			middleware.ValidateSession(
				st,
				middleware.CSRF(
					middleware.Auth(handler),
				),
			),
		),
	)
}
