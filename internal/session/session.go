package session

import "net/http"

type Session struct {
	Data      map[string]string
	IP        string
	UserAgent string
}

func newSession(r *http.Request) Session {
	return Session{
		Data:      make(map[string]string),
		IP:        getClientIP(r),
		UserAgent: r.UserAgent(),
	}
}

func (s *Session) Get(key string) (string, bool) {
	value, ok := s.Data[key]
	return value, ok
}

func (s *Session) Set(key, value string) {
	s.Data[key] = value
}

func (s *Session) Delete(key string) {
	delete(s.Data, key)
}

func (s *Session) Has(key string) bool {
	_, ok := s.Data[key]
	return ok
}
