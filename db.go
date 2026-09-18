package main

import (
	"database/sql"
	"errors"

	"github.com/zeldojov/gopost/internal/store"
	_ "modernc.org/sqlite"
)

const (
	dbPath = "app.db"
)

func InitDB() (*store.Store, error) {
	if dbPath == "" {
		return nil, errors.New("dbPath is empty")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	store := store.NewStore(db)

	if err := store.CreateUsersTable(); err != nil {
		db.Close()
		return nil, err
	}

	if err := store.CreateSessionsTable(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}
