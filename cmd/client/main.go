package main

import (
	"bytes"
	"conrobb/tunnel-rat/internal/model"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	apiURL        = "https://your-api.example.com/get-config"
	configPath    = "/etc/wireguard/wg0.conf"
	clientEnv     = "CLIENT_SECRET"
	checkInterval = 30 * time.Second
)

// APIResponse represents the expected response from the central API
type APIResponse struct {
	PeerPublicKey string `json:"peer_public_key"`
	Endpoint      string `json:"endpoint"`
	AllowedIPs    string `json:"allowed_ips"`
	DNS           string `json:"dns,omitempty"`
}

func runCommand(cmd string, args ...string) (string, error) {
	output, err := exec.Command(cmd, args...).CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

func main() {
	// Get client secret from environment
	clientSecret := os.Getenv(clientEnv)
	if clientSecret == "" {
		log.Fatal("[ERROR] CLIENT_SECRET is not set")
	}

	// Generate WireGuard keypair
	privateKey, err := runCommand("wg", "genkey")
	if err != nil {
		log.Fatalf("[ERROR] Failed to generate private key: %v", err)
	}

	publicKey, err := runCommand("bash", "-c", fmt.Sprintf("echo '%s' | wg pubkey", privateKey))
	if err != nil {
		log.Fatalf("[ERROR] Failed to generate public key: %v", err)
	}

	// Send public key to central API
	requestBody, _ := json.Marshal(map[string]string{
		"client_secret": clientSecret,
		"public_key":    publicKey,
	})

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Fatalf("[ERROR] Could not create API request: %v", err)
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("[ERROR] No response from API: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("[ERROR] Could not read body")
	}

	// Parse API response
	var relayConfig model.Relay
	if err := json.Unmarshal(body, &relayConfig); err != nil {
		log.Fatalf("[ERROR] Failed to parse API response: %v", err)
	}

	// Write WireGuard configuration
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("[Interface]\nPrivateKey = %s\nAddress = 10.0.0.2/24\n", privateKey))
	if relayConfig.DNS != "" {
		buffer.WriteString(fmt.Sprintf("DNS = %s\n", relayConfig.DNS))
	}
	buffer.WriteString(fmt.Sprintf("[Peer]\nPublicKey = %s\nEndpoint = %s\nAllowedIPs = %s\nPersistentKeepalive = 25\n",
		relayConfig.PublicKey, relayConfig.Endpoint, relayConfig.AllowedIPs))

	if err := os.WriteFile(configPath, buffer.Bytes(), 0600); err != nil {
		log.Fatalf("[ERROR] Failed to write WireGuard config: %v", err)
	}

	log.Println("[INFO] WireGuard configuration written.")

	// Start WireGuard
	if _, err := runCommand("wg-quick", "up", "wg0"); err != nil {
		log.Fatalf("[ERROR] Failed to start WireGuard: %v", err)
	}

	log.Println("[INFO] WireGuard tunnel established.")

	// Monitor connection status
	for {
		time.Sleep(checkInterval)
		if _, err := runCommand("wg", "show", "wg0"); err != nil {
			log.Println("[WARNING] WireGuard tunnel appears down. Restarting...")
			runCommand("wg-quick", "down", "wg0")
			runCommand("wg-quick", "up", "wg0")
		} else {
			log.Println("[INFO] WireGuard tunnel is active.")
		}
	}
}
