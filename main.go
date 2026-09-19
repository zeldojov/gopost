package main

import (
	"log"
	"net/http"

	"github.com/zeldojov/gopost/internal/handlers"
	"github.com/zeldojov/gopost/internal/middleware"
	"github.com/zeldojov/gopost/internal/store"

	_ "github.com/go-sql-driver/mysql"
)

const (
	dbPath = "app.db"
)

func main() {

	db, err := store.Connect("root", "root", "127.0.0.1", "3306", "gozex")
	if err != nil {
		log.Fatal(err)
	}

	store, err := store.NewStore(db)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	mux := http.DefaultServeMux

	h := handlers.NewHandler(store)

	http.Handle("POST /login", middleware.GuestChain(store, http.HandlerFunc(h.Login)))
	http.Handle("GET /login/", middleware.GuestChain(store, http.HandlerFunc(h.LoginPage)))

	http.Handle("POST /register", middleware.GuestChain(store, http.HandlerFunc(h.Register)))
	http.Handle("GET /register/", middleware.GuestChain(store, http.HandlerFunc(h.RegisterPage)))

	http.Handle("GET /user/home", middleware.AuthChain(store, http.HandlerFunc(h.UserHome)))
	http.Handle("POST /logout", middleware.AuthChain(store, http.HandlerFunc(h.Logout)))

	handler := middleware.PublicChain(store, mux)

	if err := http.ListenAndServe("127.0.0.1:8000", handler); err != nil {
		log.Fatal(err)
	}

}
