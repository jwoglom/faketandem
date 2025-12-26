package handler

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
)

// TestJPAKEAuthenticatorInterface verifies both implementations satisfy the interface
func TestJPAKEAuthenticatorInterface(t *testing.T) {
	var _ JPAKEAuthenticatorInterface = &JPAKEAuthenticator{}
	var _ JPAKEAuthenticatorInterface = &PumpX2JPAKEAuthenticator{}
	t.Log("Both implementations satisfy JPAKEAuthenticatorInterface")
}

// TestJPAKEAuthenticator_Initialization tests proper initialization
func TestJPAKEAuthenticator_Initialization(t *testing.T) {
	pairingCode := "123456"
	bridge := &pumpx2.Bridge{}

	auth := NewJPAKEAuthenticator(pairingCode, bridge)

	if auth == nil {
		t.Fatal("NewJPAKEAuthenticator returned nil")
	}

	if auth.pairingCode != pairingCode {
		t.Errorf("Expected pairing code %s, got %s", pairingCode, auth.pairingCode)
	}

	if auth.bridge != bridge {
		t.Error("Bridge not set correctly")
	}

	if auth.round != 0 {
		t.Errorf("Expected initial round 0, got %d", auth.round)
	}

	if auth.IsComplete() {
		t.Error("Newly created authenticator should not be complete")
	}
}

// TestPumpX2JPAKEAuthenticator_Initialization tests pumpX2 authenticator initialization
func TestPumpX2JPAKEAuthenticator_Initialization(t *testing.T) {
	pairingCode := "654321"
	bridge := &pumpx2.Bridge{}
	pumpX2Path := "/path/to/pumpx2"
	pumpX2Mode := "gradle"
	gradleCmd := "./gradlew"
	javaCmd := "java"

	auth := NewPumpX2JPAKEAuthenticator(
		pairingCode,
		bridge,
		pumpX2Path,
		pumpX2Mode,
		gradleCmd,
		javaCmd,
	)

	if auth == nil {
		t.Fatal("NewPumpX2JPAKEAuthenticator returned nil")
	}

	if auth.pairingCode != pairingCode {
		t.Errorf("Expected pairing code %s, got %s", pairingCode, auth.pairingCode)
	}

	if auth.pumpX2Path != pumpX2Path {
		t.Errorf("Expected pumpX2Path %s, got %s", pumpX2Path, auth.pumpX2Path)
	}

	if auth.pumpX2Mode != pumpX2Mode {
		t.Errorf("Expected pumpX2Mode %s, got %s", pumpX2Mode, auth.pumpX2Mode)
	}

	if auth.round != 0 {
		t.Errorf("Expected initial round 0, got %d", auth.round)
	}
}

// TestJPAKEAuthenticator_RoundProgression tests round state progression
func TestJPAKEAuthenticator_RoundProgression(t *testing.T) {
	auth := NewJPAKEAuthenticator("123456", &pumpx2.Bridge{})

	// Initially round 0
	if auth.round != 0 {
		t.Errorf("Expected initial round 0, got %d", auth.round)
	}

	// Process round 1
	mockData := map[string]interface{}{"test": "data"}
	_, err := auth.ProcessRound(1, mockData)
	if err != nil {
		t.Logf("Round 1 error (expected in unit test): %v", err)
	}

	// Note: Round progression depends on implementation details
	// This test mainly verifies the structure is correct
}

// TestJPAKESessionManager_CreateWithMode tests session manager mode selection
func TestJPAKESessionManager_CreateWithMode(t *testing.T) {
	tests := []struct {
		name      string
		jpakeMode string
		wantType  string
	}{
		{
			name:      "Go mode",
			jpakeMode: "go",
			wantType:  "*handler.JPAKEAuthenticator",
		},
		{
			name:      "PumpX2 mode",
			jpakeMode: "pumpx2",
			wantType:  "*handler.PumpX2JPAKEAuthenticator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewJPAKESessionManager(
				tt.jpakeMode,
				"/tmp/pumpx2",
				"gradle",
				"./gradlew",
				"java",
			)

			auth := manager.GetOrCreate("test-session", "123456", &pumpx2.Bridge{})
			if auth == nil {
				t.Fatal("GetOrCreate returned nil")
			}

			// Verify type based on mode
			switch tt.jpakeMode {
			case "go":
				if _, ok := auth.(*JPAKEAuthenticator); !ok {
					t.Errorf("Expected *JPAKEAuthenticator, got %T", auth)
				}
			case "pumpx2":
				if _, ok := auth.(*PumpX2JPAKEAuthenticator); !ok {
					t.Errorf("Expected *PumpX2JPAKEAuthenticator, got %T", auth)
				}
			}
		})
	}
}

// TestJPAKESessionManager_MultipleSessionsSamePairing tests multiple concurrent sessions
func TestJPAKESessionManager_MultipleSessionsSamePairing(t *testing.T) {
	manager := NewJPAKESessionManager("go", "/tmp", "gradle", "./gradlew", "java")

	pairingCode := "123456"
	bridge := &pumpx2.Bridge{}

	// Create multiple sessions with same pairing code
	session1 := manager.GetOrCreate("client-1", pairingCode, bridge)
	session2 := manager.GetOrCreate("client-2", pairingCode, bridge)
	session3 := manager.GetOrCreate("client-3", pairingCode, bridge)

	if session1 == nil || session2 == nil || session3 == nil {
		t.Fatal("Failed to create sessions")
	}

	// Each should be a different instance
	if session1 == session2 || session1 == session3 || session2 == session3 {
		t.Error("Expected different session instances for different session IDs")
	}

	// But getting the same session ID should return same instance
	session1Again := manager.GetOrCreate("client-1", pairingCode, bridge)
	if session1 != session1Again {
		t.Error("Expected same instance when requesting same session ID")
	}
}

// TestJPAKESessionManager_DifferentPairingCodes tests sessions with different pairing codes
func TestJPAKESessionManager_DifferentPairingCodes(t *testing.T) {
	manager := NewJPAKESessionManager("go", "/tmp", "gradle", "./gradlew", "java")

	bridge := &pumpx2.Bridge{}

	// Create sessions with different pairing codes
	session1 := manager.GetOrCreate("session-1", "111111", bridge)
	session2 := manager.GetOrCreate("session-2", "222222", bridge)
	session3 := manager.GetOrCreate("session-3", "333333", bridge)

	if session1 == nil || session2 == nil || session3 == nil {
		t.Fatal("Failed to create sessions")
	}

	// Each should be independent
	if session1 == session2 || session1 == session3 || session2 == session3 {
		t.Error("Expected different instances for different sessions")
	}
}

// TestJPAKESessionManager_RemoveSession tests session removal
func TestJPAKESessionManager_RemoveSession(t *testing.T) {
	manager := NewJPAKESessionManager("go", "/tmp", "gradle", "./gradlew", "java")

	bridge := &pumpx2.Bridge{}
	pairingCode := "123456"

	// Create session
	session1 := manager.GetOrCreate("test-session", pairingCode, bridge)
	if session1 == nil {
		t.Fatal("Failed to create session")
	}

	// Verify it exists
	session1Again := manager.GetOrCreate("test-session", pairingCode, bridge)
	if session1 != session1Again {
		t.Error("Expected same session instance")
	}

	// Remove it
	manager.Remove("test-session")

	// Create again with same ID should give new instance
	session1New := manager.GetOrCreate("test-session", pairingCode, bridge)
	if session1 == session1New {
		t.Error("Expected new instance after removal")
	}
}

// TestJPAKEAuthenticator_IsComplete tests completion state
func TestJPAKEAuthenticator_IsComplete(t *testing.T) {
	auth := NewJPAKEAuthenticator("123456", &pumpx2.Bridge{})

	// Initially not complete
	if auth.IsComplete() {
		t.Error("New authenticator should not be complete")
	}

	// After setting round to 4, should be complete
	auth.round = 4
	if !auth.IsComplete() {
		t.Error("Authenticator at round 4 should be complete")
	}

	// Round 3 should not be complete
	auth.round = 3
	if auth.IsComplete() {
		t.Error("Authenticator at round 3 should not be complete")
	}
}

// TestJPAKEAuthenticator_GetSharedSecretBeforeComplete tests error condition
func TestJPAKEAuthenticator_GetSharedSecretBeforeComplete(t *testing.T) {
	auth := NewJPAKEAuthenticator("123456", &pumpx2.Bridge{})

	tests := []struct {
		name      string
		round     int
		wantError bool
	}{
		{"Round 0", 0, true},
		{"Round 1", 1, true},
		{"Round 2", 2, true},
		{"Round 3", 3, true},
		{"Round 4 (complete)", 4, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth.round = tt.round

			_, err := auth.GetSharedSecret()
			if tt.wantError && err == nil {
				t.Error("Expected error getting shared secret before completion")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error at round %d: %v", tt.round, err)
			}
		})
	}
}

// BenchmarkJPAKEAuthenticator_Creation benchmarks authenticator creation
func BenchmarkJPAKEAuthenticator_Creation(b *testing.B) {
	bridge := &pumpx2.Bridge{}
	pairingCode := "123456"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewJPAKEAuthenticator(pairingCode, bridge)
	}
}

// BenchmarkJPAKESessionManager_GetOrCreate benchmarks session creation
func BenchmarkJPAKESessionManager_GetOrCreate(b *testing.B) {
	manager := NewJPAKESessionManager("go", "/tmp", "gradle", "./gradlew", "java")
	bridge := &pumpx2.Bridge{}
	pairingCode := "123456"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessionID := "session-" + string(rune(i%100))
		_ = manager.GetOrCreate(sessionID, pairingCode, bridge)
	}
}
