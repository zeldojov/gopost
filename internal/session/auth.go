package session

import (
	"net/http"
	"uuid"
)

func (s *Session) IsAuthenticated() bool {
	return s.userID != nil
}

func (s *Session) IsAnonymous() bool {
	return s.userID == nil
}

func (s *Session) Authenticate(userID uuid.UUID, r *http.Request) {
	*s = *NewAuthSession(userID, r)
}
