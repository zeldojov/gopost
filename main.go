package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/zeldojov/gopost/internal/handlers"
	"github.com/zeldojov/gopost/internal/middleware"
	"github.com/zeldojov/gopost/internal/session"
)

var LOG = log.Default()
var DB *sql.DB
var err error

func init() {
	if err = InitDB(); err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	session.DB = DB
}

func main() {
	defer DB.Close()

	mux := http.DefaultServeMux

	mux.Handle("GET /login", middleware.Guest(http.HandlerFunc(handlers.LoginPage)))
	mux.Handle("POST /login", middleware.Guest(http.HandlerFunc(handlers.Login)))

	mux.Handle("GET /register", middleware.Guest(http.HandlerFunc(handlers.RegisterPage)))
	mux.Handle("POST /register", middleware.Guest(http.HandlerFunc(handlers.Register)))

	mux.Handle("GET /user/home", middleware.Auth(http.HandlerFunc(handlers.UserHome)))
	mux.Handle("POST /logout", middleware.Auth(http.HandlerFunc(handlers.Logout)))

	handler := middleware.AllowedMethod(
		middleware.Session(
			middleware.ValidateSession(
				middleware.CSRF(mux),
			),
		),
	)

	if err := http.ListenAndServe("127.0.0.1:8000", handler); err != nil {
		LOG.Fatal(err)
	}

}
