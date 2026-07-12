package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"
)

// TestQuickReconnectJPAKEAuthenticator_FullExchange drives a full round 3/4
// exchange the way a real quick-pairing client would, independently
// recomputing the expected session key and HMACs (mirroring pumpX2's
// Hkdf.build/HmacSha256.hmacSha256, verified against the real pumpX2 source),
// to guard against a regression in the hand-ported crypto.
func TestQuickReconnectJPAKEAuthenticator_FullExchange(t *testing.T) {
	longTermSecret := []byte("0123456789abcdef0123456789abcdef")
	auth := NewQuickReconnectJPAKEAuthenticator(longTermSecret)

	if auth.IsComplete() {
		t.Fatal("freshly created quick-reconnect authenticator should not be complete")
	}

	sessionKey := quickReconnectDriveRound3(t, auth, longTermSecret)
	quickReconnectDriveRound4(t, auth, sessionKey)

	if !auth.IsComplete() {
		t.Fatal("authenticator should be complete after round 4")
	}

	sharedSecret, err := auth.GetSharedSecret()
	if err != nil {
		t.Fatalf("GetSharedSecret failed after completion: %v", err)
	}
	if string(sharedSecret) != string(longTermSecret) {
		t.Error("GetSharedSecret should return the cached long-term secret")
	}

	longTerm, err := auth.GetLongTermSecret()
	if err != nil {
		t.Fatalf("GetLongTermSecret failed: %v", err)
	}
	if string(longTerm) != string(longTermSecret) {
		t.Error("GetLongTermSecret should return the cached long-term secret unchanged")
	}
}

// quickReconnectDriveRound3 processes round 3 and independently recomputes
// the expected session key (Hkdf.build(serverNonce, longTermSecret)).
func quickReconnectDriveRound3(t *testing.T, auth *QuickReconnectJPAKEAuthenticator, longTermSecret []byte) []byte {
	t.Helper()

	round3Resp, err := auth.ProcessRound(3, nil)
	if err != nil {
		t.Fatalf("round 3 failed: %v", err)
	}

	serverNonceHex, ok := round3Resp["nonce"].(string)
	if !ok || serverNonceHex == "" {
		t.Fatalf("expected non-empty nonce in round 3 response, got %v", round3Resp)
	}
	serverNonce, err := hex.DecodeString(serverNonceHex)
	if err != nil {
		t.Fatalf("round 3 nonce is not valid hex: %v", err)
	}
	if len(serverNonce) != 8 {
		t.Errorf("expected 8-byte server nonce, got %d bytes", len(serverNonce))
	}

	step1 := hmacSum(serverNonce, longTermSecret)
	return hmacSum(step1, []byte{0x01})
}

// quickReconnectDriveRound4 builds a valid client key-confirmation request
// from sessionKey and verifies the server's round 4 response against it.
func quickReconnectDriveRound4(t *testing.T, auth *QuickReconnectJPAKEAuthenticator, sessionKey []byte) {
	t.Helper()

	clientNonce := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	clientHash := hmacSum(sessionKey, clientNonce)

	round4Resp, err := auth.ProcessRound(4, map[string]interface{}{
		"nonce":      hex.EncodeToString(clientNonce),
		"hashDigest": hex.EncodeToString(clientHash),
	})
	if err != nil {
		t.Fatalf("round 4 failed: %v", err)
	}

	serverHashHex, ok := round4Resp["hashDigest"].(string)
	if !ok {
		t.Fatalf("expected hashDigest in round 4 response, got %v", round4Resp)
	}
	serverHash, err := hex.DecodeString(serverHashHex)
	if err != nil {
		t.Fatalf("round 4 hashDigest is not valid hex: %v", err)
	}
	serverNonce4Hex, _ := round4Resp["nonce"].(string)
	serverNonce4, err := hex.DecodeString(serverNonce4Hex)
	if err != nil {
		t.Fatalf("round 4 nonce is not valid hex: %v", err)
	}
	expectedServerHash := hmacSum(sessionKey, serverNonce4)
	if !hmac.Equal(serverHash, expectedServerHash) {
		t.Error("server's round 4 hashDigest does not match independently computed HMAC")
	}
}

// TestQuickReconnectJPAKEAuthenticator_BadClientHMAC guards against accepting
// a round 4 request whose HMAC doesn't match (e.g. the client's cached secret
// is stale relative to ours).
func TestQuickReconnectJPAKEAuthenticator_BadClientHMAC(t *testing.T) {
	auth := NewQuickReconnectJPAKEAuthenticator([]byte("long-term-secret"))

	if _, err := auth.ProcessRound(3, nil); err != nil {
		t.Fatalf("round 3 failed: %v", err)
	}

	_, err := auth.ProcessRound(4, map[string]interface{}{
		"nonce":      hex.EncodeToString([]byte{1, 2, 3, 4, 5, 6, 7, 8}),
		"hashDigest": hex.EncodeToString(make([]byte, 32)), // wrong HMAC
	})
	if err == nil {
		t.Error("expected an error for a mismatched client HMAC")
	}
}

// TestQuickReconnectJPAKEAuthenticator_OutOfOrder guards against answering
// round 4 before round 3 has run.
func TestQuickReconnectJPAKEAuthenticator_OutOfOrder(t *testing.T) {
	auth := NewQuickReconnectJPAKEAuthenticator([]byte("long-term-secret"))

	_, err := auth.ProcessRound(4, map[string]interface{}{
		"nonce":      hex.EncodeToString([]byte{1, 2, 3, 4, 5, 6, 7, 8}),
		"hashDigest": hex.EncodeToString(make([]byte, 32)),
	})
	if err == nil {
		t.Error("expected an error when round 4 is requested before round 3")
	}
}

// TestJPAKESessionManager_QuickPairWithCachedKey verifies that a fresh
// session receiving Jpake3SessionKeyRequest as its first message is resumed
// via QuickReconnectJPAKEAuthenticator when a long-term key is cached.
func TestJPAKESessionManager_QuickPairWithCachedKey(t *testing.T) {
	pumpState := state.NewPumpState()
	pumpState.SetLongTermKey([]byte("cached-long-term-secret"))

	manager := NewJPAKESessionManager("pumpx2", "/tmp/pumpx2", "gradle", "./gradlew", "java", "", pumpState)

	auth, err := manager.GetOrCreate("default", "123456", &pumpx2.Bridge{}, 3)
	if err != nil {
		t.Fatalf("expected quick-pair to be honored, got error: %v", err)
	}
	if _, ok := auth.(*QuickReconnectJPAKEAuthenticator); !ok {
		t.Errorf("expected *QuickReconnectJPAKEAuthenticator, got %T", auth)
	}
}

// TestJPAKESessionManager_QuickPairWithoutCachedKeyRejected verifies that a
// quick-pair attempt with no cached long-term key is rejected rather than
// silently spawning pumpX2's jpake-server for round 3 (which requires rounds
// 1a/1b/2 to have already run against that process and would fail outright).
func TestJPAKESessionManager_QuickPairWithoutCachedKeyRejected(t *testing.T) {
	manager := NewJPAKESessionManager("pumpx2", "/tmp/pumpx2", "gradle", "./gradlew", "java", "", state.NewPumpState())

	_, err := manager.GetOrCreate("default", "123456", &pumpx2.Bridge{}, 3)
	if !errors.Is(err, ErrJPAKEQuickPairRejected) {
		t.Errorf("expected ErrJPAKEQuickPairRejected, got %v", err)
	}
}

func hmacSum(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}
