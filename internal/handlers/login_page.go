package handlers

import (
	"html/template"
	"log"
	"net/http"

	"github.com/zeldojov/gopost/internal/session"
)

func LoginPage(w http.ResponseWriter, r *http.Request) {
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

	{{if .Error}}
		<p>{{.Error}}</p>
	{{end}}

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

	sess, ok := session.GetSession(r)
	if !ok {
		log.Printf("session missing from request")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	errorMessage, _ := sess.GetFlash("error")

	if err := tmpl.Execute(w, struct {
		CSRFToken string
		Error     string
	}{
		CSRFToken: sess.CSRFToken(),
		Error:     errorMessage,
	}); err != nil {
		log.Printf("failed to render login page: %v", err)
	}
}
