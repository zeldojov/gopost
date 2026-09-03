package session

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"maps"
	"net"
	"net/http"
	"sync"
)

var ErrSessionNotFound = errors.New("session not found")

type Store struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

func NewStore() *Store {
	return &Store{
		sessions: make(map[string]Session),
	}
}

func newSessionID() string {
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

func (s *Store) AddSession(r *http.Request) string {
	sessionID := newSessionID()
	session := newSession(r)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[sessionID] = session

	return sessionID
}

func (s *Store) RemoveSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[sessionID]; !ok {
		return ErrSessionNotFound
	}

	delete(s.sessions, sessionID)

	return nil
}

func (s *Store) GetSession(sessionID string) (Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return Session{}, ErrSessionNotFound
	}

	data := make(map[string]string, len(session.Data))

	maps.Copy(data, session.Data)

	return Session{
		Data:      data,
		IP:        session.IP,
		UserAgent: session.UserAgent,
	}, nil
}

func (s *Store) UpdateSession(sessionID string, session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[sessionID]; !ok {
		return ErrSessionNotFound
	}

	s.sessions[sessionID] = session

	return nil
}
