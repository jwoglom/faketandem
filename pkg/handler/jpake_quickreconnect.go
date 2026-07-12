package handler

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
)

// QuickReconnectJPAKEAuthenticator handles a "quick pair" reconnect: a client
// that was already fully paired earlier (rounds 1a/1b/2/3/4 completed once)
// reconnects and sends Jpake3SessionKeyRequest directly, skipping straight to
// round 3 instead of renegotiating the password-authenticated exchange from
// Jpake1aRequest. Real Tandem apps do this by reusing the JPAKE-derived
// long-term secret from the original pairing; this authenticator answers
// rounds 3/4 against that cached secret using the same HKDF/HMAC-SHA256
// construction pumpX2's jpake-server uses (see Hkdf.build/HmacSha256.hmacSha256
// in pumpX2's messages/builders/crypto package), without needing pumpX2's
// stateful subprocess, which can only run the full 1a->1b->2->3->4 sequence.
type QuickReconnectJPAKEAuthenticator struct {
	longTermSecret []byte

	round       int
	serverNonce []byte
	sessionKey  []byte

	mutex sync.Mutex
}

// NewQuickReconnectJPAKEAuthenticator creates an authenticator that resumes
// from a previously cached long-term secret instead of running full JPAKE.
func NewQuickReconnectJPAKEAuthenticator(longTermSecret []byte) *QuickReconnectJPAKEAuthenticator {
	return &QuickReconnectJPAKEAuthenticator{
		longTermSecret: longTermSecret,
	}
}

// ProcessRound processes JPAKE round 3 or 4 using the cached long-term secret.
func (j *QuickReconnectJPAKEAuthenticator) ProcessRound(round int, requestData map[string]interface{}) (map[string]interface{}, error) {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	switch round {
	case 3:
		return j.processRound3()
	case 4:
		return j.processRound4(requestData)
	default:
		return nil, fmt.Errorf("quick-pair reconnect cannot handle JPAKE round %d (only rounds 3/4 are supported without a full renegotiation)", round)
	}
}

// processRound3 generates a fresh server nonce and derives this connection's
// session key from the cached long-term secret via Hkdf.build(serverNonce,
// longTermSecret), matching pumpX2's jpake-server round 3 behavior.
func (j *QuickReconnectJPAKEAuthenticator) processRound3() (map[string]interface{}, error) {
	serverNonce := make([]byte, 8)
	if _, err := rand.Read(serverNonce); err != nil {
		return nil, fmt.Errorf("failed to generate server nonce: %w", err)
	}

	j.serverNonce = serverNonce
	j.sessionKey = hkdfBuild(serverNonce, j.longTermSecret)
	j.round = 3

	log.Debug("Quick-pair reconnect: answered round 3 from cached long-term key")

	return map[string]interface{}{
		"appInstanceId": 0,
		"nonce":         hex.EncodeToString(serverNonce),
		"reserved":      hex.EncodeToString(make([]byte, 8)),
	}, nil
}

// processRound4 verifies the client's key confirmation HMAC and returns our own.
func (j *QuickReconnectJPAKEAuthenticator) processRound4(requestData map[string]interface{}) (map[string]interface{}, error) {
	if j.round != 3 {
		return nil, fmt.Errorf("quick-pair JPAKE out of order: expected round 3 complete, got round %d", j.round)
	}

	clientNonceHex, _ := requestData["nonce"].(string)
	clientHashHex, _ := requestData["hashDigest"].(string)

	clientNonce, err := hex.DecodeString(clientNonceHex)
	if err != nil {
		return nil, fmt.Errorf("invalid client nonce in Jpake4KeyConfirmationRequest: %w", err)
	}
	clientHash, err := hex.DecodeString(clientHashHex)
	if err != nil {
		return nil, fmt.Errorf("invalid client hashDigest in Jpake4KeyConfirmationRequest: %w", err)
	}

	expectedClientHash := hmacSha256(j.sessionKey, clientNonce)
	if !hmac.Equal(expectedClientHash, clientHash) {
		return nil, fmt.Errorf("quick-pair client HMAC validation failed (cached long-term key no longer matches the client)")
	}

	serverNonce := make([]byte, 8)
	if _, err := rand.Read(serverNonce); err != nil {
		return nil, fmt.Errorf("failed to generate server confirmation nonce: %w", err)
	}
	serverHash := hmacSha256(j.sessionKey, serverNonce)

	j.round = 4

	log.Info("Quick-pair reconnect: JPAKE key confirmation succeeded using cached long-term key")

	return map[string]interface{}{
		"appInstanceId": 0,
		"nonce":         hex.EncodeToString(serverNonce),
		"reserved":      hex.EncodeToString(make([]byte, 8)),
		"hashDigest":    hex.EncodeToString(serverHash),
	}, nil
}

// GetSharedSecret returns the long-term secret, matching the convention
// PumpX2JPAKEAuthenticator uses for its own GetSharedSecret/AuthKey (the
// per-connection authenticated-message signing key comes from the long-term
// secret in both paths, not the ephemeral per-connection HKDF session key).
func (j *QuickReconnectJPAKEAuthenticator) GetSharedSecret() ([]byte, error) {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	if j.round < 4 {
		return nil, fmt.Errorf("quick-pair JPAKE not complete, current round: %d", j.round)
	}

	return j.longTermSecret, nil
}

// GetLongTermSecret returns the cached long-term secret this authenticator
// was resumed from, so the caller can (re-)persist it -- it never changes.
func (j *QuickReconnectJPAKEAuthenticator) GetLongTermSecret() ([]byte, error) {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	return j.longTermSecret, nil
}

// IsComplete returns true once round 4 key confirmation has succeeded.
func (j *QuickReconnectJPAKEAuthenticator) IsComplete() bool {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	return j.round == 4
}

// hkdfBuild replicates pumpX2's Hkdf.build(nonce, keyMaterial):
//
//	step1 := HMAC-SHA256(key=nonce, data=keyMaterial)
//	return HMAC-SHA256(key=step1, data=0x01)
//
// (a single-block expansion producing exactly 32 bytes, since Hkdf.build
// always requests a 32-byte key -- see pumpX2's
// messages/builders/crypto/Hkdf.java: build() calls
// expandKey(updateMacWithKeyMaterial(nonce, keyMaterial), new byte[0], 32),
// and expandKey's loop runs a single iteration for a 32-byte keyLength,
// hashing counter byte 0x01 with no additional nonce/salt.)
func hkdfBuild(nonce, keyMaterial []byte) []byte {
	step1 := hmacSha256(nonce, keyMaterial)
	return hmacSha256(step1, []byte{0x01})
}

// hmacSha256 computes HMAC-SHA256(key, data), matching pumpX2's
// HmacSha256.hmacSha256(data, key) (its mod255 byte-sign massaging is a no-op
// for Go's unsigned byte slices, so this is a direct HMAC-SHA256 call).
func hmacSha256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}
