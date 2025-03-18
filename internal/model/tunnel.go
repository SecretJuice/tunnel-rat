package model

import "time"

type TunnelStatus string

const (
	PENDING     TunnelStatus = "pending"
	ACTIVE      TunnelStatus = "active"
	TERMINATING TunnelStatus = "terminating"
	TERMINATED  TunnelStatus = "terminated"
)

type Tunnel struct {
	ID              string       `json:"tunnel_id"`
	Status          TunnelStatus `json:"status"`
	RelayID         string       `json:"relay_id"`
	ClientID        string       `json:"client_id"`
	ClientPubKey    string       `json:"client_pub_key"`
	RequestTime     time.Time    `json:"request_time"`
	EstablishedTime time.Time    `json:"established_time"`
	TerminatedTime  time.Time    `json:"terminated_time"`
}

func CreateTunnel(client Client) Tunnel {
	return Tunnel{}
}
