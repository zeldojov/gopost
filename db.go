package main

import (
	"database/sql"
	"errors"

	_ "modernc.org/sqlite"
)

const (
	dbPath = "app.db"
)

func InitDB() error {
	if dbPath == "" {
		return errors.New("dbPath is empty")
	}

	sqlite, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	if err := sqlite.Ping(); err != nil {
		sqlite.Close()
		return err
	}

	if err := createUsersTable(sqlite); err != nil {
		sqlite.Close()
		return err
	}

	if _, err := sqlite.Exec(createSessionsTableQuery); err != nil {
		sqlite.Close()
		return err
	}

	DB = sqlite

	return nil
}
