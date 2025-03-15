package model

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
)

func generateSecret(length int) string {
	// Calculate the number of random bytes needed
	numBytes := (length * 3) / 4 // Base64 encoding expands data, so we adjust accordingly

	// Generate random bytes
	bytes := make([]byte, numBytes)
	rand.Read(bytes)

	// Encode to base64 URL format (URL-safe: A-Z, a-z, 0-9, '-', '_')
	str := base64.URLEncoding.EncodeToString(bytes)

	// Remove padding and ensure correct length
	str = strings.TrimRight(str, "=")
	if len(str) > length {
		str = str[:length]
	}

	return str
}

type Client struct {
	PublicKey  string `json:"public_key"`
	AllowedIPs string `json:"allowed_ips"`
	DNS        string `json:"dns,omitempty"`
	Secret     string `json:"client_secret"`
}

var clientStore map[string]Client = make(map[string]Client) //string=Client Secret

func ValidateSecret(secret string) bool {
	for key := range clientStore {
		if key == secret {
			return true
		}
	}
	return false
}

func CreateClient(client Client) (string, error) {
	if client.Secret == "" {
		client.Secret = generateSecret(48)
	}

	clientStore[client.Secret] = client

	return client.Secret, nil
}
