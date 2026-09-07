package session

import (
	"errors"
	"net/http"
	"time"

	database "github.com/zeldojov/gopost/internal/database"
)

type Store struct {
	repository database.SessionRepositoryInterface
}

func NewStore(repository database.SessionRepositoryInterface) *Store {
	return &Store{
		repository: repository,
	}
}

func (s *Store) GetSession(sessionID string) (Session, error) {
	record, err := s.repository.GetSession(sessionID)

	if errors.Is(err, database.ErrNotFound) {
		return Session{}, ErrSessionNotFound
	}

	if err != nil {
		return Session{}, err
	}

	if time.Now().After(record.ExpiresAt) {
		return Session{}, ErrSessionExpired
	}

	return Session{
		Data:      record.Data,
		IP:        record.IP,
		UserAgent: record.UserAgent,
		CreatedAt: record.CreatedAt,
		ExpiresAt: record.ExpiresAt,
	}, nil
}

func (s *Store) AddSession(r *http.Request) (string, error) {
	sessionID := newSessionID()
	sess := newSession(r)

	err := s.repository.CreateSession(database.Session{
		ID:        sessionID,
		Data:      sess.Data,
		IP:        sess.IP,
		UserAgent: sess.UserAgent,
		CreatedAt: sess.CreatedAt,
		ExpiresAt: sess.ExpiresAt,
	})
	if err != nil {
		return "", err
	}

	return sessionID, nil
}

func (s *Store) RemoveSession(sessionID string) error {
	err := s.repository.DeleteSession(sessionID)

	if errors.Is(err, database.ErrNotFound) {
		return ErrSessionNotFound
	}

	return err
}

func (s *Store) UpdateSession(sessionID string, sess Session) error {
	err := s.repository.UpdateSession(database.Session{
		ID:        sessionID,
		Data:      sess.Data,
		IP:        sess.IP,
		UserAgent: sess.UserAgent,
		CreatedAt: sess.CreatedAt,
		ExpiresAt: sess.ExpiresAt,
	})

	if errors.Is(err, database.ErrNotFound) {
		return ErrSessionNotFound
	}

	return err
}

func (s *Store) RefreshSession(sessionID string) (Session, error) {
	sess, err := s.GetSession(sessionID)
	if err != nil {
		return Session{}, err
	}

	sess.ExpiresAt = time.Now().Add(sessionDuration)

	if err := s.UpdateSession(sessionID, sess); err != nil {
		return Session{}, err
	}

	return sess, nil
}

func (s *Store) RegenerateSessionID(sessionID string) (string, error) {
	sess, err := s.GetSession(sessionID)
	if err != nil {
		return "", err
	}

	newID := newSessionID()
	now := time.Now()

	err = s.repository.RegenerateSession(
		sessionID,
		database.Session{
			ID:        newID,
			Data:      sess.Data,
			IP:        sess.IP,
			UserAgent: sess.UserAgent,
			CreatedAt: now,
			ExpiresAt: now.Add(sessionDuration),
		},
	)

	if errors.Is(err, database.ErrNotFound) {
		return "", ErrSessionNotFound
	}

	if err != nil {
		return "", err
	}

	return newID, nil
}

func (s *Store) NeedsRegeneration(sessionID string) (bool, error) {
	sess, err := s.GetSession(sessionID)
	if err != nil {
		return false, err
	}

	return time.Since(sess.CreatedAt) >= sessionAbsoluteDuration, nil
}

func (s *Store) MatchesRequest(sessionID string, r *http.Request) (bool, error) {
	sess, err := s.GetSession(sessionID)
	if err != nil {
		return false, err
	}

	return sess.IP == getClientIP(r) &&
		sess.UserAgent == r.UserAgent(), nil
}

func (s *Store) createSession(w http.ResponseWriter, r *http.Request, config CookieConfig) (string, error) {
	sessionID, err := s.AddSession(r)
	if err != nil {
		return "", err
	}

	SetSessionCookie(w, config, sessionID)

	return sessionID, nil
}
