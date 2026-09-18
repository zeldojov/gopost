package session

import (
	"context"
	"net/http"
)

func GetSession(r *http.Request) (*Session, bool) {
	sess, ok := r.Context().Value(ContextKey{}).(*Session)
	return sess, ok
}

func SetSession(sess *Session, r *http.Request) *http.Request {
	ctx := context.WithValue(r.Context(), ContextKey{}, sess)
	return r.WithContext(ctx)
}
