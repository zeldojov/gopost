package main

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
)

var LOG = log.Default()
var DB *sql.DB
var err error

func init() {
	if err = InitDB(); err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
}

func main() {
	defer DB.Close()

	mux := http.DefaultServeMux

	mux.HandleFunc("GET /login", LoginPageHandler)
	mux.HandleFunc("POST /login", LoginHandler)
	mux.HandleFunc("GET /register", RegisterPageHandler)
	mux.HandleFunc("POST /register", RegisterHandler)
	mux.Handle(
		"GET /user/home",
		AuthMiddleware(http.HandlerFunc(UserHomeHandler)),
	)

	handler := AllowedMethodMiddleware(
		SessionMiddleware(
			ValidateSessionMiddleware(
				CSRFMiddleware(mux),
			),
		),
	)

	if err := http.ListenAndServe("127.0.0.1:8000", handler); err != nil {
		LOG.Fatal(err)
	}

}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

	if err := validateUsername(username); err != nil {
		http.Error(w, "invalid username", http.StatusBadRequest)
		return
	}

	if err := validatePassword(password); err != nil {
		http.Error(w, "invalid password", http.StatusBadRequest)
		return
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		LOG.Printf("failed to hash password: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user := CreateUser(username, passwordHash)

	if err := user.Save(); err != nil {
		if errors.Is(err, errUsernameTaken) {
			http.Error(w, "username already exists", http.StatusConflict)
			return
		}

		LOG.Printf("failed to save user: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	sess, ok := GetSession(r)
	if !ok {
		LOG.Printf("session missing from request")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := sess.Authenticate(user.ID, w, r); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

	user, err := GetUserByUsername(username)
	if err != nil {
		if errors.Is(err, errUserNotFound) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}

		LOG.Printf("failed to get user: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if !verifyPassword(password, user.PasswordHash) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	sess, ok := GetSession(r)
	if !ok {
		LOG.Printf("session missing from request")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := sess.Authenticate(user.ID, w, r); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/user/home", http.StatusSeeOther)
}

func LoginPageHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("login").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Login</title>
</head>
<body>
	<h1>Login</h1>

	<form method="POST" action="/login">
		<label>
			Username:
			<input type="text" name="username" required>
		</label>

		<br><br>

		<label>
			Password:
			<input type="password" name="password" required>
		</label>

		<br><br>

		<input type="hidden" name="csrf_token" value="{{.CSRFToken}}">

		<button type="submit">Login</button>
	</form>

	<p>
		Don't have an account?
		<a href="/register">Register</a>
	</p>
</body>
</html>
`))

	sess, ok := GetSession(r)
	if !ok {
		LOG.Printf("session missing from request")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, struct {
		CSRFToken string
	}{
		CSRFToken: sess.csrfToken,
	}); err != nil {
		LOG.Printf("failed to render login page: %v", err)
	}
}

func RegisterPageHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("register").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Register</title>
</head>
<body>
	<h1>Register</h1>

	<form method="POST" action="/register">
		<label>
			Username:
			<input type="text" name="username" required>
		</label>

		<br><br>

		<label>
			Password:
			<input type="password" name="password" required>
		</label>

		<br><br>

		<input type="hidden" name="csrf_token" value="{{.CSRFToken}}">

		<button type="submit">Register</button>
	</form>

	<p>
		Already have an account?
		<a href="/login">Login</a>
	</p>
</body>
</html>
`))

	sess, ok := GetSession(r)
	if !ok {
		LOG.Printf("session missing from request")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, struct {
		CSRFToken string
	}{
		CSRFToken: sess.csrfToken,
	}); err != nil {
		LOG.Printf("failed to render register page: %v", err)
	}
}

func UserHomeHandler(w http.ResponseWriter, r *http.Request) {
	LOG.Println("=================== UserHomeHandler ========================")
	sess, ok := GetSession(r)
	if !ok {
		LOG.Printf("session missing from request")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := GetUserByID(*sess.userID)
	if err != nil {
		if errors.Is(err, errUserNotFound) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		LOG.Printf("failed to get user: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Welcome, %s!", user.Username)
}
