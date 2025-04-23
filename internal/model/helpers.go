package model

import (
	"database/sql"
	"log/slog"
)

type ModelContext struct {
	Logger  *slog.Logger
	Db      *sql.DB
	Clients ClientModel
	Relays  RelayModel
	Tunnels TunnelModel
}
