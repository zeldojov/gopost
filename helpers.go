package main

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
)

func NewRandomToken() string {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return hex.EncodeToString(b)
}

func GetClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

func GetUserAgent(r *http.Request) string {
	return r.UserAgent()
}

func GetSession(r *http.Request) (*Session, bool) {
	sess, ok := r.Context().Value(contextKey{}).(*Session)
	return sess, ok
}

func GetCSRFToken(r *http.Request) string {
	session, ok := GetSession(r)
	if !ok || session == nil {
		return ""
	}

	if session.csrfToken == "" {
		session.csrfToken = NewRandomToken()
	}

	return session.csrfToken
}
