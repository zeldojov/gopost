package main

import (
	"database/sql"

	"github.com/zeldojov/gopost/internal/session"
)

func initDB(path string) (*sql.DB, error) {
	db, err := session.OpenDB(path)
	if err != nil {
		return nil, err
	}

	if err := session.CreateSessionsTable(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
