package utils

import (
	"net"
	"net/http"
)

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
