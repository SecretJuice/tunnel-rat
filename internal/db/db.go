package db

import (
	"database/sql"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const createUserTable = `
CREATE TABLE IF NOT EXISTS "user"(
    "id" SERIAL PRIMARY KEY NOT NULL,
    "username" VARCHAR(255) NOT NULL
);
`

const createClientTable = `
CREATE TABLE IF NOT EXISTS "client"(
    "id" SERIAL PRIMARY KEY NOT NULL,
    "name" VARCHAR(255) NOT NULL,
    "subdomain" VARCHAR(255) UNIQUE NOT NULL,
    "region" VARCHAR(255) CHECK
        ("region" IN('na-w', 'na-e')) NOT NULL,
        "public_key" VARCHAR(255) NULL,
        "secret" VARCHAR(255) UNIQUE NOT NULL,
        "user" BIGINT NOT NULL REFERENCES "user"("id"),
        "relay" BIGINT NULL REFERENCES "relay"("id")
);
`

const createRelayTable = `
CREATE TABLE IF NOT EXISTS "relay"(
    "id" SERIAL PRIMARY KEY NOT NULL,
    "region" VARCHAR(255) CHECK
        ("region" IN('')) NOT NULL,
        "active" BOOLEAN NOT NULL,
        "public_key" VARCHAR(255) NOT NULL,
        "ip_address" VARCHAR(255) NOT NULL
);
`

const createPortMappingTable = `
CREATE TABLE IF NOT EXISTS "port_mapping"(
    "id" SERIAL PRIMARY KEY NOT NULL,
    "client" BIGINT NOT NULL REFERENCES "client"("id"),
    "port" VARCHAR(255) NOT NULL,
    "protocol" VARCHAR(255) CHECK
        ("protocol" IN('')) NOT NULL
);
`

const sessionsTable = `
CREATE TABLE IF NOT EXISTS sessions (
    token TEXT PRIMARY KEY,
    data BYTEA NOT NULL,
    expiry TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions (expiry);
`

func EnsureTables(db *sql.DB, logger *slog.Logger) error {
	logger.Debug("Checking for missing tables...")

	_, err := db.Exec(createUserTable)
	if err != nil {
		logger.Error("Could not create user table", "error", err.Error())
		return err
	}
	_, err = db.Exec(createRelayTable)
	if err != nil {
		logger.Error("Could not create relay table", "error", err.Error())
		return err
	}
	_, err = db.Exec(createClientTable)
	if err != nil {
		logger.Error("Could not create client table", "error", err.Error())
		return err
	}
	_, err = db.Exec(createPortMappingTable)
	if err != nil {
		logger.Error("Could not create port_mapping table", "error", err.Error())
		return err
	}
	_, err = db.Exec(sessionsTable)
	if err != nil {
		logger.Error("Could not create sessions table", "error", err.Error())
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
