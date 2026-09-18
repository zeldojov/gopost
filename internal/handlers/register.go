package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/zeldojov/gopost/internal/session"
	"github.com/zeldojov/gopost/internal/user"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	sess, ok := session.GetSession(r)
	if !ok {
		log.Printf("session missing from request")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

	if err := user.ValidateUsername(username); err != nil {
		sess.SetFlash("error", "invalid username")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	if err := user.ValidatePassword(password); err != nil {
		sess.SetFlash("error", "invalid password")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	passwordHash, err := user.HashPassword(password)
	if err != nil {
		log.Printf("failed to hash password: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	newUser := user.CreateUser(username, passwordHash)

	if err := h.store.SaveUser(newUser); err != nil {
		if errors.Is(err, user.ErrUsernameTaken) {
			sess.SetFlash("error", "username already exists")
			http.Redirect(w, r, "/register", http.StatusSeeOther)
			return
		}

		log.Printf("failed to save user: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Session fixation protection:
	// delete old anonymous session and create a new authenticated one.
	if err := h.store.DeleteSession(sess); err != nil {
		log.Printf("failed to delete old session: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	session.UnsetSessionCookie(w)

	sess.Authenticate(newUser.ID, r)

	if err := h.store.SaveSession(sess); err != nil {
		log.Printf("failed to save authenticated session: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	session.SetSessionCookie(w, sess.ID())

	http.Redirect(w, r, "/user/home", http.StatusSeeOther)
}
