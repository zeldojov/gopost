package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/zeldojov/gopost/internal/app"
	"github.com/zeldojov/gopost/internal/session"
)

var (
	config app.Config
	db     *sql.DB
)

func init() {
	config = app.Config{
		Address: "127.0.0.1:8000",
	}

	var err error

	db, err = session.OpenDB("data/gopost.db")
	if err != nil {
		log.Fatal(err)
	}

	if err := session.CreateSessionsTable(db); err != nil {
		log.Fatal(err)
	}
}

func main() {
	config := app.Config{
		DBPath:  "data/gopost.db",
		Address: "127.0.0.1:8000",
	}

	a, err := app.New(config)
	if err != nil {
		log.Fatal(err)
	}
	defer a.Close()

	mux := http.NewServeMux()

	handler := app.Chain(
		a.MethodMiddleware,
		a.SessionMiddleware,
		a.CSRFMiddleware,
	)(mux)

	if err := a.Run(handler); err != nil {
		log.Fatal(err)
	}
}
