package session

import (
	"net/http"

	"github.com/zeldojov/gopost/internal/utils"
)

const (
	csrfFieldName = "csrf_token"
)

func CSRFFieldName() string {
	return csrfFieldName
}

func GetCSRFToken(r *http.Request) string {
	session, ok := GetSession(r)
	if !ok || session == nil {
		return ""
	}

	if session.csrfToken == "" {
		session.csrfToken = utils.NewRandomToken()
	}

	return session.csrfToken
}

func (s *Session) ValidateCSRFToken(r *http.Request) bool {
	if s.csrfToken == "" {
		return false
	}

	requestToken := r.PostFormValue(csrfFieldName)

	return requestToken != "" &&
		requestToken == s.csrfToken
}
