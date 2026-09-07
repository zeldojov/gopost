package main

import (
	"log"
	"net/http"

	"github.com/zeldojov/gopost/internal/app"
)

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
