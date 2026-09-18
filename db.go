package main

import (
	"database/sql"
	"errors"

	"github.com/zeldojov/gopost/internal/session"
	"github.com/zeldojov/gopost/internal/user"
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

	if err := user.CreateUsersTable(sqlite); err != nil {
		sqlite.Close()
		return err
	}

	if err := session.CreateSessionsTable(sqlite); err != nil {
		sqlite.Close()
		return err
	}

	DB = sqlite

	return nil
}
