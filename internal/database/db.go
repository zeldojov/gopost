package database

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func OpenDB(path string) (*sql.DB, error) {
	return sql.Open("sqlite", path)
}

const createSessionsTable = `
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    csrf_token TEXT NOT NULL,
    user_id TEXT NULL,
    user_ip TEXT NOT NULL,
    user_agent TEXT NOT NULL,
    user_data TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL
);
`

func CreateSessionsTable(db *sql.DB) error {
	_, err := db.Exec(createSessionsTable)
	return err
}
