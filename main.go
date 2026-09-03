package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/zeldojov/gopost/internal/session"
)

type PageData struct {
	Title string
	Name  string
}

func main() {

	store := session.NewStore()

	tmpl := template.Must(template.ParseFiles("templates/index.html"))

	mux := http.NewServeMux()
	handler := session.Middleware(store)(mux)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		sess, ok := session.Get(r)
		if !ok {
			http.Error(w, "session not found", http.StatusInternalServerError)
			return
		}

		sess.Set("user_id", "zeljko")

		data := PageData{
			Title: "My Go App",
			Name:  "Željko",
		}

		w.WriteHeader(http.StatusOK)
		tmpl.Execute(w, data)
	})

	log.Print("Running server on http://127.0.0.1:8000")
	if err := http.ListenAndServe("127.0.0.1:8000", handler); err != nil {
		log.Fatal("Fatal error running a http server.")
	}

}
