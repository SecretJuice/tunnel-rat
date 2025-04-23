package model

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

type Region string

const (
	NA_WEST Region = "na-w"
	NA_EAST Region = "na-e"
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

const letters = "abcdefghijklmnopqrstuvwxyz"

func generateSubdomain(n int) (string, error) {
	b := make([]byte, n)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		b[i] = letters[num.Int64()]
	}
	return string(b), nil
}

type Client struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Subdomain  string `json:"subdomain"`
	Region     Region `json:"region"`
	PublicKey  string `json:"public_key"`
	AllowedIPs string `json:"allowed_ips"`
	DNS        string `json:"dns,omitempty"`
	Secret     string `json:"client_secret"`
	UserID     int64  `json:"user_id"`
	Relay      int64  `json:"relay_id"`
}

type ClientModel struct {
	*ModelContext
}

var (
	ErrNotFound error = errors.New("not found in database")
)

func (m *ClientModel) UpdatePublicKey(clientId int64, pubkey string) error {
	query := `
		UPDATE client
		SET public_key = $1
		WHERE id = $2
		RETURNING public_key	
	`
	result, err := m.Db.Exec(
		query,
		pubkey,
		clientId,
	)

	affected, err := result.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}

	if err != nil {
		return err
	}

	return nil
}

func (m *ClientModel) GetBySecret(secret string) (*Client, error) {
	query := `
		SELECT id, name, subdomain, region, secret, "user", relay FROM client
		WHERE secret=$1;
	`
	var client Client

	var relayId sql.NullInt64

	err := m.Db.QueryRow(
		query,
		secret,
	).Scan(
		&client.ID,
		&client.Name,
		&client.Subdomain,
		&client.Region,
		&client.Secret,
		&client.UserID,
		&relayId,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if relayId.Valid {
		client.Relay = relayId.Int64
	}
	return &client, nil

}

func (m *ClientModel) CreateClient(userId int64, name string, region Region) (Client, error) {
	query := `
        INSERT INTO client (name, subdomain, region, secret, "user")
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id;
	`
	subdomain, _ := generateSubdomain(5)
	secret := generateSecret(60)

	var clientID int64
	err := m.Db.QueryRow(
		query,
		name,
		subdomain,
		region,
		secret,
		userId,
	).Scan(&clientID)

	client := Client{
		ID:        clientID,
		Name:      name,
		Subdomain: subdomain,
		Region:    region,
		Secret:    secret,
		UserID:    userId,
	}

	if err != nil {
		return Client{}, fmt.Errorf("failed to create client: %w", err)
	}

	return client, nil
}
