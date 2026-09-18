package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/zeldojov/gopost/internal/session"
	"github.com/zeldojov/gopost/internal/user"
)

func Login(w http.ResponseWriter, r *http.Request) {
	sess, ok := session.GetSession(r)
	if !ok {
		log.Printf("session missing from request")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

	foundUser, err := user.GetUserByUsername(username)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			sess.SetFlash("error", "invalid credentials")
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		log.Printf("failed to get user: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !user.VerifyPassword(password, foundUser.PasswordHash) {
		sess.SetFlash("error", "invalid credentials")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := sess.Authenticate(foundUser.ID, w, r); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/user/home", http.StatusSeeOther)
}
