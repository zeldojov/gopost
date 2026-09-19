package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
)

var (
	errDatabaseOpen = errors.New("failed to open sqlite database connection")
	errDatabasePing = errors.New("failed to ping sqlite database connection")
	errDBStoreInit  = errors.New("failed to initialize database store")
)

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
	dsn := NewDSN(username, password, address, port, database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errDatabaseOpen, err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		log.Fatalf("%v: %v", errDatabasePing, err)
		return nil, fmt.Errorf("%w: %v", errDatabasePing, err)
	}

	return db, nil
}
