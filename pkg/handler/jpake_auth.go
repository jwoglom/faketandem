package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"

	"github.com/jwoglom/faketandem/pkg/pumpx2"

	log "github.com/sirupsen/logrus"
)

// JPAKEAuthenticator manages JPAKE authentication state for a connection
type JPAKEAuthenticator struct {
	pairingCode string
	bridge      *pumpx2.Bridge

	// JPAKE state
	round          int
	clientID       string
	serverID       string
	sharedSecret   []byte

	// Crypto parameters (simplified for now)
	x2             *big.Int // Server's secret
	gx1, gx2       []byte   // Server's public values
	gx3, gx4       []byte   // Client's public values (from round 1)

	mutex          sync.Mutex
}

// NewJPAKEAuthenticator creates a new JPAKE authenticator
func NewJPAKEAuthenticator(pairingCode string, bridge *pumpx2.Bridge) *JPAKEAuthenticator {
	return &JPAKEAuthenticator{
		pairingCode: pairingCode,
		bridge:      bridge,
		round:       0,
		serverID:    "tandem-pump",
		clientID:    "pumpx2-client",
	}
}

// ProcessRound processes a JPAKE round and returns the response parameters
func (j *JPAKEAuthenticator) ProcessRound(round int, requestData map[string]interface{}) (map[string]interface{}, error) {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	log.Infof("Processing JPAKE round %d", round)

	switch round {
	case 1:
		return j.processRound1(requestData)
	case 2:
		return j.processRound2(requestData)
	case 3:
		return j.processRound3(requestData)
	case 4:
		return j.processRound4(requestData)
	default:
		return nil, fmt.Errorf("invalid JPAKE round: %d", round)
	}
}

// processRound1 handles the first JPAKE round
// Client sends gx3, gx4 (their ephemeral public keys)
// Server responds with gx1, gx2 (our ephemeral public keys)
func (j *JPAKEAuthenticator) processRound1(requestData map[string]interface{}) (map[string]interface{}, error) {
	log.Debug("JPAKE Round 1: Generating server ephemeral keys")

	// Extract client's public values from request
	// In a real implementation, these would be parsed from the cargo
	if gx3, ok := requestData["gx3"].(string); ok {
		j.gx3, _ = hex.DecodeString(gx3)
	}
	if gx4, ok := requestData["gx4"].(string); ok {
		j.gx4, _ = hex.DecodeString(gx4)
	}

	// Generate server's ephemeral keys
	// For now, using simplified crypto - in production this would use proper JPAKE
	j.x2 = generateRandom256Bit()
	j.gx1 = generatePublicValue(j.x2)
	j.gx2 = generatePublicValue(j.x2)

	j.round = 1

	// Return response parameters
	response := map[string]interface{}{
		"gx1":            hex.EncodeToString(j.gx1),
		"gx2":            hex.EncodeToString(j.gx2),
		"zkpX1":          generateZKP(j.gx1), // Zero-knowledge proof for gx1
		"zkpX2":          generateZKP(j.gx2), // Zero-knowledge proof for gx2
		"serverIdentity": j.serverID,
	}

	log.Debugf("JPAKE Round 1 response generated: gx1=%d bytes, gx2=%d bytes",
		len(j.gx1), len(j.gx2))

	return response, nil
}

// processRound2 handles the second JPAKE round
// Client sends A (combined public key)
// Server responds with B (our combined public key)
func (j *JPAKEAuthenticator) processRound2(requestData map[string]interface{}) (map[string]interface{}, error) {
	log.Debug("JPAKE Round 2: Computing shared key material")

	if j.round != 1 {
		return nil, fmt.Errorf("JPAKE out of order: expected round 1 complete, got round %d", j.round)
	}

	// Extract client's A value
	var clientA []byte
	if a, ok := requestData["a"].(string); ok {
		clientA, _ = hex.DecodeString(a)
	}

	// Compute server's B value using pairing code
	// B = g^((x1+x2+x3)*s) where s is derived from pairing code
	pairingSecret := derivePairingSecret(j.pairingCode)
	serverB := computeCombinedKey(j.gx1, j.gx2, clientA, pairingSecret)

	j.round = 2

	response := map[string]interface{}{
		"b":    hex.EncodeToString(serverB),
		"zkpB": generateZKP(serverB),
	}

	log.Debugf("JPAKE Round 2 response generated: B=%d bytes", len(serverB))

	return response, nil
}

// processRound3 handles the third JPAKE round
// Client sends their key confirmation
// Server responds with our key confirmation
func (j *JPAKEAuthenticator) processRound3(requestData map[string]interface{}) (map[string]interface{}, error) {
	log.Debug("JPAKE Round 3: Key confirmation")

	if j.round != 2 {
		return nil, fmt.Errorf("JPAKE out of order: expected round 2 complete, got round %d", j.round)
	}

	// Extract client's key confirmation MAC
	var clientMac []byte
	if mac, ok := requestData["mac"].(string); ok {
		clientMac, _ = hex.DecodeString(mac)
	}

	// Compute shared secret
	pairingSecret := derivePairingSecret(j.pairingCode)
	j.sharedSecret = deriveSharedSecret(j.gx1, j.gx2, j.gx3, j.gx4, pairingSecret)

	// Generate server's key confirmation MAC
	serverMac := computeKeyConfirmationMAC(j.sharedSecret, j.serverID, clientMac)

	j.round = 3

	response := map[string]interface{}{
		"mac": hex.EncodeToString(serverMac),
	}

	log.Debugf("JPAKE Round 3 response generated: MAC=%d bytes, shared secret derived",
		len(serverMac))

	return response, nil
}

// processRound4 handles the fourth and final JPAKE round
// Server confirms authentication is complete
func (j *JPAKEAuthenticator) processRound4(requestData map[string]interface{}) (map[string]interface{}, error) {
	log.Debug("JPAKE Round 4: Authentication complete")

	if j.round != 3 {
		return nil, fmt.Errorf("JPAKE out of order: expected round 3 complete, got round %d", j.round)
	}

	// Verify client's final confirmation if present
	if finalMac, ok := requestData["finalMac"].(string); ok {
		clientFinalMac, _ := hex.DecodeString(finalMac)
		if !verifyFinalMAC(j.sharedSecret, clientFinalMac) {
			return nil, fmt.Errorf("JPAKE final MAC verification failed")
		}
	}

	j.round = 4

	// Return success with derived key
	response := map[string]interface{}{
		"status":     "authenticated",
		"derivedKey": hex.EncodeToString(j.sharedSecret),
	}

	log.Infof("JPAKE authentication completed successfully, shared secret: %d bytes",
		len(j.sharedSecret))

	return response, nil
}

// GetSharedSecret returns the derived shared secret (only valid after round 4)
func (j *JPAKEAuthenticator) GetSharedSecret() ([]byte, error) {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	if j.round < 4 {
		return nil, fmt.Errorf("JPAKE not complete, current round: %d", j.round)
	}

	return j.sharedSecret, nil
}

// IsComplete returns true if JPAKE authentication is complete
func (j *JPAKEAuthenticator) IsComplete() bool {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	return j.round == 4
}

// Helper functions for simplified JPAKE crypto
// NOTE: These are simplified implementations for demonstration
// Production should use proper JPAKE library or pumpX2's implementation

func generateRandom256Bit() *big.Int {
	max := new(big.Int)
	max.Exp(big.NewInt(2), big.NewInt(256), nil)
	n, _ := rand.Int(rand.Reader, max)
	return n
}

func generatePublicValue(secret *big.Int) []byte {
	// Simplified: In real JPAKE, this would be g^x mod p
	hash := sha256.Sum256(secret.Bytes())
	return hash[:]
}

func generateZKP(publicValue []byte) string {
	// Simplified zero-knowledge proof
	// Real implementation would use Schnorr signatures
	hash := sha256.Sum256(publicValue)
	return hex.EncodeToString(hash[:16])
}

func derivePairingSecret(pairingCode string) []byte {
	// Derive secret from pairing code
	hash := sha256.Sum256([]byte(pairingCode))
	return hash[:]
}

func computeCombinedKey(gx1, gx2, clientA, pairingSecret []byte) []byte {
	// Simplified combined key computation
	combined := append(gx1, gx2...)
	combined = append(combined, clientA...)
	combined = append(combined, pairingSecret...)
	hash := sha256.Sum256(combined)
	return hash[:]
}

func deriveSharedSecret(gx1, gx2, gx3, gx4, pairingSecret []byte) []byte {
	// Derive the final shared secret
	combined := append(gx1, gx2...)
	combined = append(combined, gx3...)
	combined = append(combined, gx4...)
	combined = append(combined, pairingSecret...)

	// Double hash for extra security
	hash1 := sha256.Sum256(combined)
	hash2 := sha256.Sum256(hash1[:])
	return hash2[:]
}

func computeKeyConfirmationMAC(sharedSecret []byte, identity string, clientMac []byte) []byte {
	// Compute HMAC-style key confirmation
	combined := append(sharedSecret, []byte(identity)...)
	combined = append(combined, clientMac...)
	hash := sha256.Sum256(combined)
	return hash[:16] // Use first 16 bytes as MAC
}

func verifyFinalMAC(sharedSecret, clientMac []byte) bool {
	// Verify client's final MAC
	// Simplified verification - in production, compare against expected HMAC
	_ = sha256.Sum256(sharedSecret) // Would use this in real implementation
	return len(clientMac) > 0
}
