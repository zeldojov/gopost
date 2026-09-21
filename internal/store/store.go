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

	errDatabaseOpen = errors.New("failed to open mysql database connection")
	errDatabasePing = errors.New("failed to ping mysql database connection")
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}

	if _, err := s.db.Exec(createUsersTableQuery); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateUsersTable, err)
	}

	if _, err := s.db.Exec(createSessionsTableQuery); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateSessionsTable, err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func NewDSN(username, password, address, port, database string) string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true",
		username,
		password,
		address,
		port,
		database,
	)
}

func Connect(username, password, address, port, database string) (*sql.DB, error) {
	dsn := NewDSN(
		username,
		password,
		address,
		port,
		database,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errDatabaseOpen, err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("%w: %w", errDatabasePing, err)
	}

	return db, nil
}
