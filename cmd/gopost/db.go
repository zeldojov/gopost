package main

import (
	"database/sql"

	"github.com/zeldojov/gopost/internal/database"
)

func initDB(path string) (*sql.DB, error) {
	db, err := database.OpenDB(path)
	if err != nil {
		return nil, err
	}

	if err := database.CreateSessionsTable(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
