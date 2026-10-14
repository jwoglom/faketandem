package handler

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"
)

// TestJPAKEAuthenticatorInterface verifies both implementations satisfy the interface
func TestJPAKEAuthenticatorInterface(t *testing.T) {
	var _ JPAKEAuthenticatorInterface = &JPAKEAuthenticator{}
	var _ JPAKEAuthenticatorInterface = &PumpX2JPAKEAuthenticator{}
	t.Log("Both implementations satisfy JPAKEAuthenticatorInterface")
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
		"",
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
				"",
				state.NewPumpState(),
			)

			auth, err := manager.GetOrCreate("test-session", "123456", &pumpx2.Bridge{}, 1)
			if err != nil {
				t.Fatalf("GetOrCreate returned error: %v", err)
			}
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
	manager := NewJPAKESessionManager("go", "/tmp", "gradle", "./gradlew", "java", "", state.NewPumpState())

	pairingCode := "123456"
	bridge := &pumpx2.Bridge{}

	// Create multiple sessions with same pairing code
	session1, err := manager.GetOrCreate("client-1", pairingCode, bridge, 1)
	if err != nil {
		t.Fatalf("GetOrCreate returned error: %v", err)
	}
	session2, err := manager.GetOrCreate("client-2", pairingCode, bridge, 1)
	if err != nil {
		t.Fatalf("GetOrCreate returned error: %v", err)
	}
	session3, err := manager.GetOrCreate("client-3", pairingCode, bridge, 1)
	if err != nil {
		t.Fatalf("GetOrCreate returned error: %v", err)
	}

	if session1 == nil || session2 == nil || session3 == nil {
		t.Fatal("Failed to create sessions")
	}

	// Each should be a different instance
	if session1 == session2 || session1 == session3 || session2 == session3 {
		t.Error("Expected different session instances for different session IDs")
	}

	// But getting the same session ID should return same instance
	session1Again, err := manager.GetOrCreate("client-1", pairingCode, bridge, 1)
	if err != nil {
		t.Fatalf("GetOrCreate returned error: %v", err)
	}
	if session1 != session1Again {
		t.Error("Expected same instance when requesting same session ID")
	}
}

// TestJPAKESessionManager_DifferentPairingCodes tests sessions with different pairing codes
func TestJPAKESessionManager_DifferentPairingCodes(t *testing.T) {
	manager := NewJPAKESessionManager("go", "/tmp", "gradle", "./gradlew", "java", "", state.NewPumpState())

	bridge := &pumpx2.Bridge{}

	// Create sessions with different pairing codes
	session1, err := manager.GetOrCreate("session-1", "111111", bridge, 1)
	if err != nil {
		t.Fatalf("GetOrCreate returned error: %v", err)
	}
	session2, err := manager.GetOrCreate("session-2", "222222", bridge, 1)
	if err != nil {
		t.Fatalf("GetOrCreate returned error: %v", err)
	}
	session3, err := manager.GetOrCreate("session-3", "333333", bridge, 1)
	if err != nil {
		t.Fatalf("GetOrCreate returned error: %v", err)
	}

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
	manager := NewJPAKESessionManager("go", "/tmp", "gradle", "./gradlew", "java", "", state.NewPumpState())

	bridge := &pumpx2.Bridge{}
	pairingCode := "123456"

	// Create session
	session1, err := manager.GetOrCreate("test-session", pairingCode, bridge, 1)
	if err != nil {
		t.Fatalf("GetOrCreate returned error: %v", err)
	}
	if session1 == nil {
		t.Fatal("Failed to create session")
	}

	// Verify it exists
	session1Again, err := manager.GetOrCreate("test-session", pairingCode, bridge, 1)
	if err != nil {
		t.Fatalf("GetOrCreate returned error: %v", err)
	}
	if session1 != session1Again {
		t.Error("Expected same session instance")
	}

	// Remove it
	manager.Remove("test-session")

	// Create again with same ID should give new instance
	session1New, err := manager.GetOrCreate("test-session", pairingCode, bridge, 1)
	if err != nil {
		t.Fatalf("GetOrCreate returned error: %v", err)
	}
	if session1 == session1New {
		t.Error("Expected new instance after removal")
	}
}

// BenchmarkJPAKEAuthenticator_Creation benchmarks authenticator creation
func BenchmarkJPAKEAuthenticator_Creation(b *testing.B) {
	pairingCode := "123456"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewJPAKEAuthenticator(pairingCode)
	}
}

// BenchmarkJPAKESessionManager_GetOrCreate benchmarks session creation
func BenchmarkJPAKESessionManager_GetOrCreate(b *testing.B) {
	manager := NewJPAKESessionManager("go", "/tmp", "gradle", "./gradlew", "java", "", state.NewPumpState())
	bridge := &pumpx2.Bridge{}
	pairingCode := "123456"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessionID := "session-" + string(rune(i%100))
		_, _ = manager.GetOrCreate(sessionID, pairingCode, bridge, 1)
	}
}
