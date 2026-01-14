package handler

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
)

// TestPumpX2JPAKEAuthenticator_FullFlow tests the complete JPAKE authentication flow
// by running pumpX2's jpake client against jpake-server in a true end-to-end test.
// This validates that actual JPAKE pairing works with real cryptographic handshaking.
//
//nolint:gocyclo // Integration test with complex bidirectional communication
func TestPumpX2JPAKEAuthenticator_FullFlow(t *testing.T) {
	// Skip if PUMPX2_PATH not set
	pumpX2Path := os.Getenv("PUMPX2_PATH")
	if pumpX2Path == "" {
		t.Skip("Skipping pumpX2 JPAKE test: PUMPX2_PATH environment variable not set")
	}

	// Verify pumpX2 directory exists
	if _, err := os.Stat(pumpX2Path); os.IsNotExist(err) {
		t.Fatalf("pumpX2 directory does not exist: %s", pumpX2Path)
	}

	// Setup logging
	log.SetLevel(log.DebugLevel)

	// Test pairing code - must match on both sides
	pairingCode := "123456"

	// Get the script path for gradle mode
	repoRoot := getRepoRoot()
	scriptPath := filepath.Join(repoRoot, "scripts", "cliparser-gradle.sh")

	// Start jpake-server (acts as pump/server)
	serverCmd := exec.Command(scriptPath, pumpX2Path, "jpake-server", pairingCode)
	serverStdin, err := serverCmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to create server stdin pipe: %v", err)
	}
	serverStdout, err := serverCmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create server stdout pipe: %v", err)
	}
	serverCmd.Stderr = os.Stderr
	defer func() {
		_ = serverStdin.Close()
		if serverCmd.Process != nil {
			_ = serverCmd.Process.Kill()
		}
	}()

	// Start jpake client (acts as mobile app/client)
	clientCmd := exec.Command(scriptPath, pumpX2Path, "jpake", pairingCode)
	clientStdin, err := clientCmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to create client stdin pipe: %v", err)
	}
	clientStdout, err := clientCmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create client stdout pipe: %v", err)
	}
	clientCmd.Stderr = os.Stderr
	defer func() {
		_ = clientStdin.Close()
		if clientCmd.Process != nil {
			_ = clientCmd.Process.Kill()
		}
	}()

	t.Log("Starting JPAKE server (pump simulator)...")
	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start jpake-server: %v", err)
	}

	t.Log("Starting JPAKE client (mobile app simulator)...")
	if err := clientCmd.Start(); err != nil {
		t.Fatalf("Failed to start jpake client: %v", err)
	}

	// Create buffered readers
	serverReader := bufio.NewReader(serverStdout)
	clientReader := bufio.NewReader(clientStdout)

	// Track results
	var serverDerivedSecret, clientDerivedSecret string
	var clientValidated bool
	derivedSecretRegex := regexp.MustCompile(`"derivedSecret"\s*:\s*"([a-fA-F0-9]+)"`)

	// Channels for communication between goroutines
	serverLines := make(chan string, 100)
	clientLines := make(chan string, 100)
	errChan := make(chan error, 2)

	// Read server output
	go func() {
		for {
			line, err := serverReader.ReadString('\n')
			if err != nil {
				errChan <- err
				return
			}
			line = strings.TrimSpace(line)
			if line != "" {
				serverLines <- line
			}
		}
	}()

	// Read client output
	go func() {
		for {
			line, err := clientReader.ReadString('\n')
			if err != nil {
				errChan <- err
				return
			}
			line = strings.TrimSpace(line)
			if line != "" {
				clientLines <- line
			}
		}
	}()

	// Process JPAKE handshake with timeout
	timeout := time.After(90 * time.Second)
	done := false

	for !done {
		select {
		case line := <-serverLines:
			t.Logf("SERVER: %s", line)

			// Check for derived secret
			if matches := derivedSecretRegex.FindStringSubmatch(line); len(matches) > 1 {
				serverDerivedSecret = matches[1]
				t.Logf("Server derived secret: %s", serverDerivedSecret)
				// Don't exit yet - wait for client to also complete
			}

			// Extract server JPAKE responses and forward to client
			// Server outputs: JPAKE_1A: {json}, JPAKE_1B: {json}, JPAKE_2: {json}, etc.
			// Send packets space-separated on one line for the parser
			if strings.Contains(line, "JPAKE_") && strings.Contains(line, "packets") {
				// Extract JSON from line
				colonIdx := strings.Index(line, ": ")
				if colonIdx > 0 {
					jsonStr := line[colonIdx+2:]
					var jsonData map[string]interface{}
					if err := json.Unmarshal([]byte(jsonStr), &jsonData); err == nil {
						// Build space-separated packet list
						if packets, ok := jsonData["packets"].([]interface{}); ok && len(packets) > 0 {
							var packetStrs []string
							for _, p := range packets {
								if hexData, ok := p.(string); ok {
									packetStrs = append(packetStrs, hexData)
								}
							}
							spaceSeparated := strings.Join(packetStrs, " ")
							t.Logf("Forwarding %d server packets to client: %s...",
								len(packetStrs), spaceSeparated[:min(80, len(spaceSeparated))])
							if _, err := clientStdin.Write([]byte(spaceSeparated + "\n")); err != nil {
								t.Logf("Error writing to client: %v", err)
							}
						}
					}
				}
			}

		case line := <-clientLines:
			t.Logf("CLIENT: %s", line)

			// Check for derived secret from client (now outputs JSON with derivedSecret)
			if matches := derivedSecretRegex.FindStringSubmatch(line); len(matches) > 1 {
				clientDerivedSecret = matches[1]
				t.Logf("Client derived secret: %s", clientDerivedSecret)
				clientValidated = true
			}

			// Also check for HMAC validation message (fallback)
			if strings.Contains(line, "HMAC SECRET VALIDATES") {
				clientValidated = true
				t.Logf("Client HMAC validation successful")
			}

			// Client outputs requests as ROUND_XX_SENT: {json with packets array}
			// Send packets space-separated on one line for the parser
			if strings.Contains(line, "_SENT:") && strings.Contains(line, "packets") {
				colonIdx := strings.Index(line, ": ")
				if colonIdx > 0 {
					jsonStr := line[colonIdx+2:]
					var jsonData map[string]interface{}
					if err := json.Unmarshal([]byte(jsonStr), &jsonData); err == nil {
						// Build space-separated packet list
						if packets, ok := jsonData["packets"].([]interface{}); ok && len(packets) > 0 {
							var packetStrs []string
							for _, p := range packets {
								if hexData, ok := p.(string); ok {
									packetStrs = append(packetStrs, hexData)
								}
							}
							spaceSeparated := strings.Join(packetStrs, " ")
							t.Logf("Forwarding %d client packets to server: %s...",
								len(packetStrs), spaceSeparated[:min(80, len(spaceSeparated))])
							if _, err := serverStdin.Write([]byte(spaceSeparated + "\n")); err != nil {
								t.Logf("Error writing to server: %v", err)
							}
						}
					}
				}
			}

		case err := <-errChan:
			if err.Error() != "EOF" {
				t.Logf("I/O error: %v", err)
			}
			// One process ended - check if we got results from BOTH sides
			if serverDerivedSecret != "" && clientValidated {
				done = true
			}

		case <-timeout:
			t.Fatal("JPAKE authentication timed out after 90 seconds")
		}
	}

	// Give a moment for final output
	time.Sleep(500 * time.Millisecond)

	// Close stdin pipes to signal completion
	_ = serverStdin.Close()
	_ = clientStdin.Close()

	// Wait for processes with timeout
	waitDone := make(chan struct{})
	go func() {
		_ = serverCmd.Wait()
		_ = clientCmd.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		t.Log("Both processes completed")
	case <-time.After(5 * time.Second):
		t.Log("Killing processes after timeout")
		if serverCmd.Process != nil {
			_ = serverCmd.Process.Kill()
		}
		if clientCmd.Process != nil {
			_ = clientCmd.Process.Kill()
		}
	}

	// Verify results
	if serverDerivedSecret == "" {
		t.Error("Server did not derive a shared secret")
	}
	if !clientValidated {
		t.Error("Client did not validate HMAC (authentication failed)")
	}

	// Both sides successfully authenticated - verify secrets match
	if serverDerivedSecret != "" && clientValidated {
		t.Logf("SUCCESS: JPAKE authentication completed!")
		t.Logf("Server derived secret: %s", serverDerivedSecret)
		if clientDerivedSecret != "" {
			t.Logf("Client derived secret: %s", clientDerivedSecret)
			if serverDerivedSecret == clientDerivedSecret {
				t.Log("Both sides derived the SAME shared secret!")
			} else {
				t.Errorf("MISMATCH: Server and client derived different secrets!")
			}
		} else {
			t.Log("Client HMAC validated (no derivedSecret output)")
		}
	}
}

// min returns the smaller of two ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestGoJPAKEAuthenticator_FullFlow tests the Go implementation
func TestGoJPAKEAuthenticator_FullFlow(t *testing.T) {
	// Setup logging
	log.SetLevel(log.DebugLevel)

	// Test pairing code
	pairingCode := "123456"

	// Create bridge (mock for Go implementation)
	bridge := &pumpx2.Bridge{} // Minimal bridge for testing

	// Create authenticator
	auth := NewJPAKEAuthenticator(pairingCode, bridge)

	t.Log("Testing Go JPAKE Round 1")
	clientRound1Data := generateMockJpake1aRequest()
	resp1, err := auth.ProcessRound(1, clientRound1Data)
	if err != nil {
		t.Fatalf("Round 1 failed: %v", err)
	}
	if resp1 == nil {
		t.Fatal("Round 1 response is nil")
	}
	t.Logf("Round 1 response: %+v", resp1)

	t.Log("Testing Go JPAKE Round 2")
	clientRound2Data := generateMockJpake2Request()
	resp2, err := auth.ProcessRound(2, clientRound2Data)
	if err != nil {
		t.Fatalf("Round 2 failed: %v", err)
	}
	if resp2 == nil {
		t.Fatal("Round 2 response is nil")
	}
	t.Logf("Round 2 response: %+v", resp2)

	t.Log("Testing Go JPAKE Round 3")
	clientRound3Data := generateMockJpake3Request()
	resp3, err := auth.ProcessRound(3, clientRound3Data)
	if err != nil {
		t.Fatalf("Round 3 failed: %v", err)
	}
	if resp3 == nil {
		t.Fatal("Round 3 response is nil")
	}
	t.Logf("Round 3 response: %+v", resp3)

	t.Log("Testing Go JPAKE Round 4")
	clientRound4Data := generateMockJpake4Request()
	resp4, err := auth.ProcessRound(4, clientRound4Data)
	if err != nil {
		t.Fatalf("Round 4 failed: %v", err)
	}
	if resp4 == nil {
		t.Fatal("Round 4 response is nil")
	}
	t.Logf("Round 4 response: %+v", resp4)

	// Verify authentication is complete
	if !auth.IsComplete() {
		t.Error("Authentication should be complete after round 4")
	}

	// Verify shared secret was derived
	sharedSecret, err := auth.GetSharedSecret()
	if err != nil {
		t.Fatalf("Failed to get shared secret: %v", err)
	}
	if len(sharedSecret) == 0 {
		t.Error("Shared secret is empty")
	}
	t.Logf("Shared secret length: %d bytes", len(sharedSecret))
}

// TestJPAKEAuthenticator_InvalidRound tests error handling for invalid rounds
func TestJPAKEAuthenticator_InvalidRound(t *testing.T) {
	auth := NewJPAKEAuthenticator("123456", &pumpx2.Bridge{})

	// Test invalid round number
	_, err := auth.ProcessRound(99, nil)
	if err == nil {
		t.Error("Expected error for invalid round number")
	}
	if err.Error() != "invalid JPAKE round: 99" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestJPAKEAuthenticator_IncompleteAuth tests getting shared secret before completion
func TestJPAKEAuthenticator_IncompleteAuth(t *testing.T) {
	auth := NewJPAKEAuthenticator("123456", &pumpx2.Bridge{})

	// Try to get shared secret before completing authentication
	_, err := auth.GetSharedSecret()
	if err == nil {
		t.Error("Expected error when getting shared secret before completion")
	}

	// Verify not complete
	if auth.IsComplete() {
		t.Error("Authentication should not be complete initially")
	}
}

// TestJPAKESessionManager tests session management
func TestJPAKESessionManager(t *testing.T) {
	// Create manager with Go mode
	manager := NewJPAKESessionManager("go", "/tmp/pumpx2", "gradle", "./gradlew", "java")

	// Create first session
	session1 := manager.GetOrCreate("session1", "123456", &pumpx2.Bridge{})
	if session1 == nil {
		t.Fatal("Failed to create session1")
	}

	// Get same session again
	session1Again := manager.GetOrCreate("session1", "123456", &pumpx2.Bridge{})
	if session1 != session1Again {
		t.Error("Expected to get same session instance")
	}

	// Create different session
	session2 := manager.GetOrCreate("session2", "654321", &pumpx2.Bridge{})
	if session2 == nil {
		t.Fatal("Failed to create session2")
	}
	if session1 == session2 {
		t.Error("Expected different session instances")
	}

	// Remove session
	manager.Remove("session1")

	// Create new session with same ID should be different instance
	session1New := manager.GetOrCreate("session1", "123456", &pumpx2.Bridge{})
	if session1 == session1New {
		t.Error("Expected new session instance after removal")
	}
}

// Helper functions for generating mock client data for Go JPAKE tests

func generateMockJpake1aRequest() map[string]interface{} {
	// Generate mock Jpake1aRequest data
	// In a real scenario, this would come from an actual client
	return map[string]interface{}{
		"messageName":          "Jpake1aRequest",
		"centralChallengeHash": generateRandomHex(165),
	}
}

func generateMockJpake2Request() map[string]interface{} {
	return map[string]interface{}{
		"messageName":          "Jpake2Request",
		"centralChallengeHash": generateRandomHex(165),
	}
}

func generateMockJpake3Request() map[string]interface{} {
	return map[string]interface{}{
		"messageName": "Jpake3SessionKeyRequest",
	}
}

func generateMockJpake4Request() map[string]interface{} {
	return map[string]interface{}{
		"messageName": "Jpake4KeyConfirmationRequest",
		"nonce":       generateRandomHex(8),
		"reserved":    "00000000",
		"hashDigest":  generateRandomHex(32),
	}
}

func generateRandomHex(byteLength int) string {
	// Generate random hex string of specified byte length
	bytes := make([]byte, byteLength)
	for i := range bytes {
		bytes[i] = byte(i % 256)
	}
	return hex.EncodeToString(bytes)
}
