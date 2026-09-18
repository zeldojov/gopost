package handlers

import (
	"log"
	"net/http"

	"github.com/zeldojov/gopost/internal/session"
)

func Logout(w http.ResponseWriter, r *http.Request) {
	sess, ok := session.GetSession(r)
	if !ok {
		log.Printf("session missing from request")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := sess.Destroy(); err != nil {
		log.Printf("failed to destroy session: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	session.UnsetSessionCookie(w)

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
