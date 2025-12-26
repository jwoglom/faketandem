package handler

import (
	"encoding/hex"
	"os"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
)

// TestPumpX2JPAKEAuthenticator_FullFlow tests the complete JPAKE authentication flow
// using the actual pumpX2 jpake-server command
//
//nolint:gocyclo // Integration test with multiple sequential steps
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

	// Test pairing code
	pairingCode := "123456"

	// Create bridge (needed for message parsing)
	bridge, err := createTestBridge(pumpX2Path)
	if err != nil {
		t.Fatalf("Failed to create bridge: %v", err)
	}

	// Create authenticator
	auth := NewPumpX2JPAKEAuthenticator(
		pairingCode,
		bridge,
		pumpX2Path,
		"gradle", // Use gradle mode for testing
		"./gradlew",
		"java",
	)
	defer func() {
		if err := auth.Close(); err != nil {
			t.Logf("Error closing auth: %v", err)
		}
	}()

	t.Log("Testing JPAKE Round 1a")
	// Simulate client sending Jpake1aRequest
	clientRound1aData := generateMockJpake1aRequest()
	resp1a, err := auth.ProcessRound(1, clientRound1aData)
	if err != nil {
		t.Fatalf("Round 1a failed: %v", err)
	}
	if resp1a == nil {
		t.Fatal("Round 1a response is nil")
	}
	if _, ok := resp1a["centralChallengeHash"]; !ok {
		t.Error("Round 1a response missing centralChallengeHash")
	}
	t.Logf("Round 1a response: %+v", resp1a)

	// Small delay between rounds
	time.Sleep(100 * time.Millisecond)

	t.Log("Testing JPAKE Round 1b")
	// Simulate client sending Jpake1bRequest
	clientRound1bData := generateMockJpake1bRequest()
	resp1b, err := auth.ProcessRound(1, clientRound1bData)
	if err != nil {
		t.Fatalf("Round 1b failed: %v", err)
	}
	if resp1b == nil {
		t.Fatal("Round 1b response is nil")
	}
	if _, ok := resp1b["centralChallengeHash"]; !ok {
		t.Error("Round 1b response missing centralChallengeHash")
	}
	t.Logf("Round 1b response: %+v", resp1b)

	time.Sleep(100 * time.Millisecond)

	t.Log("Testing JPAKE Round 2")
	// Simulate client sending Jpake2Request
	clientRound2Data := generateMockJpake2Request()
	resp2, err := auth.ProcessRound(2, clientRound2Data)
	if err != nil {
		t.Fatalf("Round 2 failed: %v", err)
	}
	if resp2 == nil {
		t.Fatal("Round 2 response is nil")
	}
	if _, ok := resp2["centralChallengeHash"]; !ok {
		t.Error("Round 2 response missing centralChallengeHash")
	}
	t.Logf("Round 2 response: %+v", resp2)

	time.Sleep(100 * time.Millisecond)

	t.Log("Testing JPAKE Round 3")
	// Simulate client sending Jpake3SessionKeyRequest
	clientRound3Data := generateMockJpake3Request()
	resp3, err := auth.ProcessRound(3, clientRound3Data)
	if err != nil {
		t.Fatalf("Round 3 failed: %v", err)
	}
	if resp3 == nil {
		t.Fatal("Round 3 response is nil")
	}
	if _, ok := resp3["deviceKeyNonce"]; !ok {
		t.Error("Round 3 response missing deviceKeyNonce")
	}
	if _, ok := resp3["deviceKeyReserved"]; !ok {
		t.Error("Round 3 response missing deviceKeyReserved")
	}
	t.Logf("Round 3 response: %+v", resp3)

	time.Sleep(100 * time.Millisecond)

	t.Log("Testing JPAKE Round 4")
	// Simulate client sending Jpake4KeyConfirmationRequest
	clientRound4Data := generateMockJpake4Request()
	resp4, err := auth.ProcessRound(4, clientRound4Data)
	if err != nil {
		t.Fatalf("Round 4 failed: %v", err)
	}
	if resp4 == nil {
		t.Fatal("Round 4 response is nil")
	}
	if _, ok := resp4["nonce"]; !ok {
		t.Error("Round 4 response missing nonce")
	}
	if _, ok := resp4["hashDigest"]; !ok {
		t.Error("Round 4 response missing hashDigest")
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
	t.Logf("Shared secret: %s", string(sharedSecret))
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

// Helper functions for generating mock client data

func createTestBridge(pumpX2Path string) (*pumpx2.Bridge, error) {
	// Create bridge with gradle mode
	bridge, err := pumpx2.NewBridge(pumpX2Path, "gradle", "./gradlew", "java")
	if err != nil {
		return nil, err
	}

	return bridge, nil
}

func generateMockJpake1aRequest() map[string]interface{} {
	// Generate mock Jpake1aRequest data
	// In a real scenario, this would come from an actual client
	return map[string]interface{}{
		"messageName":          "Jpake1aRequest",
		"centralChallengeHash": generateRandomHex(165),
	}
}

func generateMockJpake1bRequest() map[string]interface{} {
	return map[string]interface{}{
		"messageName":          "Jpake1bRequest",
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
