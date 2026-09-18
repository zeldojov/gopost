package main

import (
	"log"
	"net/http"

	"github.com/zeldojov/gopost/internal/handlers"
	"github.com/zeldojov/gopost/internal/middleware"
)

var LOG = log.Default()

func main() {
	store, err := InitDB()
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	mux := http.DefaultServeMux

	h := handlers.NewHandler(store)

	http.Handle("/login", middleware.GuestChain(store, http.HandlerFunc(h.Login)))
	http.Handle("/login/", middleware.GuestChain(store, http.HandlerFunc(h.LoginPage)))

	http.Handle("/register", middleware.GuestChain(store, http.HandlerFunc(h.Register)))
	http.Handle("/register/", middleware.GuestChain(store, http.HandlerFunc(h.RegisterPage)))

	http.Handle("/user/home", middleware.AuthChain(store, http.HandlerFunc(h.UserHome)))
	http.Handle("/logout", middleware.AuthChain(store, http.HandlerFunc(h.Logout)))

	handler := middleware.PublicChain(store, mux)

	if err := http.ListenAndServe("127.0.0.1:8000", handler); err != nil {
		LOG.Fatal(err)
	}

}
