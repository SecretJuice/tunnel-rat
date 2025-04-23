package model

type RelayModel struct {
	*ModelContext
}

type Relay struct {
	ID        int64  `json:"id"`
	Region    Region `json:"region"`
	PublicKey string `json:"public_key"`
	Address   string `json:"address"`  //192.168.4.1
	Endpoint  string `json:"endpoint"` //external, static ip of relay machine
	DNS       string `json:"dns,omitempty"`
}
