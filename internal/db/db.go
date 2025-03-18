package db

import (
	"database/sql"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`

func EnsureTables(db *sql.DB, logger *slog.Logger) error {
	logger.Debug("Checking for missing tables...")

	_, err := db.Exec(createUsersTable)
	if err != nil {
		logger.Error("Could not create clients table", "error", err.Error())
		return err
	}

	logger.Debug("Tables are up-to-date!")
	return nil
}

func Connect(dsn string, logger *slog.Logger) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	logger.Info("Connected to Postgres database")

	if err := EnsureTables(db, logger); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
