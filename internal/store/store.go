package store

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

var (
	ErrCreateUsersTable    = errors.New("failed to create users table")
	ErrCreateSessionsTable = errors.New("failed to create sessions table")
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}

	if _, err := s.db.Exec(createSessionsTableQuery); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCreateSessionsTable, err)
	}

	if _, err := s.db.Exec(createUsersTableQuery); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCreateUsersTable, err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
