package main

import (
	"database/sql"

	"github.com/zeldojov/gopost/internal/storage/sqlite"
)

func initDB(path string) (*sql.DB, error) {
	db, err := sqlite.OpenDB(path)
	if err != nil {
		return nil, err
	}

	if err := sqlite.CreateSessionsTable(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
