package session

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
)

func newRandomToken() string {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return hex.EncodeToString(b)
}

func getClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

func getUserAgent(r *http.Request) string {
	return r.UserAgent()
}
