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
	apiURL        = "http://host.docker.internal:8080/tunnel"
	configPath    = "/etc/wireguard/wg0.conf"
	clientEnv     = "CLIENT_SECRET"
	checkInterval = 30 * time.Second
	pollingRate   = 1 * time.Second
)

// APIResponse represents the expected response from the central API
type CreateTunnelResponse struct {
	ID int64 `json:"tunnel_id"`
}

type PeerConfig struct {
	Endpoint      string `json:"relay_endpoint"`
	PublicKey     string `json:"relay_pub_key"`
	ClientAddress string `json:"allowed_ip"`
	DNS           string `json:"dns,omitempty"`
}

func runCommand(cmd string, args ...string) (string, error) {
	output, err := exec.Command(cmd, args...).CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

func main() {
	clientSecret := os.Getenv(clientEnv)
	if clientSecret == "" {
		log.Fatal("[ERROR] CLIENT_SECRET is not set")
	}

	privateKey, err := runCommand("wg", "genkey")
	if err != nil {
		log.Fatalf("[ERROR] Failed to generate private key: %v", err)
	}

	publicKey, err := runCommand("bash", "-c", fmt.Sprintf("echo '%s' | wg pubkey", privateKey))
	if err != nil {
		log.Fatalf("[ERROR] Failed to generate public key: %v", err)
	}

	log.Println("PUBLIC KEY: " + publicKey)

	requestBody, _ := json.Marshal(map[string]string{
		"public_key": publicKey,
	})

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Fatalf("[ERROR] Could not create API request: %v", err)
	}

	req.Header.Set("Authorization", clientSecret)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("[ERROR] No response from API: %v", err)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		log.Fatalf("[ERROR] Invalid 'CLIENT_SECRET' provided")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("[ERROR] Could not read body")
	}

	var res CreateTunnelResponse

	// Parse API response
	if err := json.Unmarshal(body, &res); err != nil {
		log.Fatalf("[ERROR] Failed to parse API response: %v", err)
	}

	var peerConfig PeerConfig

	for {
		time.Sleep(pollingRate)
		config, ok := checkTunnel(res.ID, clientSecret)
		if ok {
			peerConfig = config
			break
		}
	}

	// Write WireGuard configuration
	configWg(peerConfig, privateKey)
	log.Println("[INFO] WireGuard configuration written.")

	// Start WireGuard
	startWg()
	log.Println("[INFO] WireGuard tunnel established.")

	// Monitor connection status
	for {
		time.Sleep(checkInterval)
		monitorWgConnection()
	}
}
func checkTunnel(tunnelID int64, secret string) (PeerConfig, bool) {

	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/%d", apiURL, tunnelID),
		nil,
	)
	if err != nil {
		log.Fatalf("[ERROR] Could not create API request: %v", err)
	}

	req.Header.Set("Authorization", secret)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("[ERROR] No response from API: %v", err)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		log.Fatalf("[ERROR] Invalid 'CLIENT_SECRET' provided")
	} else if resp.StatusCode != http.StatusOK {
		log.Fatalf("[ERROR] HTTP Error: %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("[ERROR] Could not read body: %v", err)
	}

	var tunnel *model.Tunnel

	if err := json.Unmarshal(body, &tunnel); err != nil {
		log.Fatalf("[ERROR] Failed to parse API response: %v", err)
	}

	log.Printf("TUNNEL: %v\n", *tunnel)

	if tunnel.Status == model.PENDING {
		return PeerConfig{}, false
	}

	peerConfig := PeerConfig{
		Endpoint:      tunnel.RelayEndpoint,
		PublicKey:     tunnel.RelayPubKey,
		ClientAddress: tunnel.AllowedIP,
	}
	return peerConfig, true
}

func monitorWgConnection() {
	if _, err := runCommand("wg", "show", "wg0"); err != nil {
		log.Println("[WARNING] WireGuard tunnel appears down. Restarting...")
		runCommand("wg-quick", "down", "wg0")
		runCommand("wg-quick", "up", "wg0")
	} else {
		log.Println("[INFO] WireGuard tunnel is active.")
	}
}
func startWg() {
	if _, err := runCommand("wg-quick", "up", "wg0"); err != nil {
		log.Fatalf("[ERROR] Failed to start WireGuard: %v", err)
	}
}
func configWg(peerConfig PeerConfig, privateKey string) {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("[Interface]\nPrivateKey = %s\nAddress = %s\n", privateKey, peerConfig.ClientAddress))
	if peerConfig.DNS != "" {
		buffer.WriteString(fmt.Sprintf("DNS = %s\n", peerConfig.DNS))
	}
	buffer.WriteString(fmt.Sprintf("[Peer]\nPublicKey = %s\nEndpoint = %s\nAllowedIPs = %s\nPersistentKeepalive = 25\n",
		peerConfig.PublicKey, peerConfig.Endpoint, "192.168.4.1/32"))

	if err := os.WriteFile(configPath, buffer.Bytes(), 0600); err != nil {
		log.Fatalf("[ERROR] Failed to write WireGuard config: %v", err)
	}
}
