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

func (sess *Session) Authenticate(userID uuid.UUID, w http.ResponseWriter, r *http.Request) error {
	if err := sess.Destroy(); err != nil {
		return err
	}

	UnsetSessionCookie(w)

	*sess = *NewAuthSession(userID, r)

	if err := sess.Save(); err != nil {
		return err
	}

	setSessionCookie(w, sess.id)

	return nil
}
