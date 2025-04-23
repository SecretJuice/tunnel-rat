package model

import (
	"database/sql"
	"errors"
	"time"
)

type TunnelStatus string

const (
	PENDING     TunnelStatus = "pending"
	ACTIVE      TunnelStatus = "active"
	TERMINATING TunnelStatus = "terminating"
	TERMINATED  TunnelStatus = "terminated"
)

type TunnelModel struct {
	*ModelContext
}

type Tunnel struct {
	ID              int64        `json:"tunnel_id"`
	Status          TunnelStatus `json:"status"`
	RelayID         int64        `json:"relay_id"`
	RelayPubKey     string       `json:"relay_pub_key"`
	RelayEndpoint   string       `json:"relay_endpoint"`
	ClientID        int64        `json:"client_id"`
	ClientPubKey    string       `json:"client_pub_key"`
	AllowedIP       string       `json:"allowed_ip"`
	RequestTime     time.Time    `json:"request_time"`
	EstablishedTime time.Time    `json:"established_time"`
	TerminatedTime  time.Time    `json:"terminated_time"`
}

func (m *TunnelModel) CreateTunnel(clientId, relayId int64) (int64, error) {
	query := `
		INSERT INTO tunnel (status, relay, client, request_time)
		VALUES ('pending', $1, $2, NOW())
		RETURNING id;
	`
	var tunnelID int64
	err := m.Db.QueryRow(
		query,
		relayId,
		clientId,
	).Scan(&tunnelID)

	if err != nil {
		return 0, err
	}

	return tunnelID, nil
}

func (m *TunnelModel) GetTunnelByClientId(clientID int64) (*Tunnel, error) {
	query := `
		SELECT t.id, status, t.relay, r.public_key, r.endpoint, t.client, c.public_key, allowed_ip, request_time,
			established_time, terminated_time FROM "tunnel" t
			JOIN "relay" r ON relay = r.id
			JOIN "client" c ON client = c.id
		WHERE client = $1 AND status != 'terminated';
	`
	var estTime sql.NullTime
	var termTime sql.NullTime
	var tunnel Tunnel
	err := m.Db.QueryRow(
		query,
		clientID,
	).Scan(
		&tunnel.ID, &tunnel.Status, &tunnel.RelayID, &tunnel.RelayPubKey, &tunnel.RelayEndpoint,
		&tunnel.AllowedIP, &tunnel.ClientID, &tunnel.ClientPubKey, &tunnel.RequestTime,
		&estTime, &termTime,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if estTime.Valid {
		tunnel.EstablishedTime = estTime.Time
	}
	if termTime.Valid {
		tunnel.TerminatedTime = termTime.Time
	}

	return &tunnel, nil
}

func (m *TunnelModel) GetById(tunnelID int64) (*Tunnel, error) {
	query := `
		SELECT t.id, status, r.id, r.public_key, r.endpoint, allowed_ip, c.id, c.public_key, request_time,
			established_time, terminated_time FROM "tunnel" t
			JOIN "relay" r ON t.relay = r.id
			JOIN "client" c ON t.client = c.id
		WHERE t.id = $1;
	`

	var estTime sql.NullTime
	var termTime sql.NullTime
	var allowedIp sql.NullString
	var tunnel Tunnel
	err := m.Db.QueryRow(
		query,
		tunnelID,
	).Scan(
		&tunnel.ID, &tunnel.Status, &tunnel.RelayID, &tunnel.RelayPubKey, &tunnel.RelayEndpoint,
		&allowedIp, &tunnel.ClientID, &tunnel.ClientPubKey, &tunnel.RequestTime,
		&estTime, &termTime,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if estTime.Valid {
		tunnel.EstablishedTime = estTime.Time
	}
	if termTime.Valid {
		tunnel.TerminatedTime = termTime.Time
	}
	if allowedIp.Valid {
		tunnel.AllowedIP = allowedIp.String
	}

	return &tunnel, nil
}
