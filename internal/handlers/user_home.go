package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/zeldojov/gopost/internal/session"
	"github.com/zeldojov/gopost/internal/user"
)

func UserHome(w http.ResponseWriter, r *http.Request) {

	sess, ok := session.GetSession(r)
	if !ok {
		log.Printf("session missing from request")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if sess.UserID() == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	foundUser, err := user.GetUserByID(*sess.UserID())
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		log.Printf("failed to get user: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
	<title>User Home</title>
</head>
<body>
	<h1>Welcome, %s!</h1>

	<form action="/logout" method="POST">
		<input type="hidden" name="csrf_token" value="%s">
		<button type="submit">Logout</button>
	</form>
</body>
</html>
`, foundUser.Username, sess.CSRFToken())
}
