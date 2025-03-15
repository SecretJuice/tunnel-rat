package model

type Relay struct {
	PublicKey  string `json:"public_key"`
	Endpoint   string `json:"endpoint"`
	AllowedIPs string `json:"allowed_ips"`
	DNS        string `json:"dns,omitempty"`
}
