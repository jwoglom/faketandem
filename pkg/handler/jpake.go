package handler

import (
	"errors"
	"fmt"
	"sync"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// JPAKEAuthenticatorInterface defines the interface for JPAKE authenticators
type JPAKEAuthenticatorInterface interface {
	ProcessRound(round int, requestData map[string]interface{}) (map[string]interface{}, error)
	GetSharedSecret() ([]byte, error)
	// GetLongTermSecret returns the raw JPAKE-derived secret suitable for
	// caching and reuse by a later quick-pair reconnect (see
	// QuickReconnectJPAKEAuthenticator). Only meaningful once IsComplete().
	GetLongTermSecret() ([]byte, error)
	IsComplete() bool
}

// ErrJPAKEQuickPairRejected is returned when a client attempts a quick-pair
// reconnect (sending Jpake3SessionKeyRequest as the first message on a fresh
// session) but the emulator has no cached long-term key to resume from --
// e.g. a fresh emulator instance that was never fully paired with this
// client before. Real pumpX2 jpake-server cannot answer round 3 without
// having run rounds 1a/1b/2 first, so the caller should force the client back
// to a full pairing by dropping the connection rather than attempting (and
// failing) to spawn pumpX2 for round 3 directly.
var ErrJPAKEQuickPairRejected = errors.New("quick-pair reconnect rejected: no cached long-term JPAKE key, client must fully re-pair")

// JPAKESessionManager manages JPAKE authentication sessions
type JPAKESessionManager struct {
	authenticators map[string]JPAKEAuthenticatorInterface
	mutex          sync.RWMutex

	// Configuration for creating authenticators
	jpakeMode     string
	pumpX2Path    string
	pumpX2Mode    string
	gradleCmd     string
	javaCmd       string
	pumpX2JarPath string

	// pumpState gives access to the cached long-term key for quick-pair
	// reconnects (see GetOrCreate).
	pumpState *state.PumpState
}

// NewJPAKESessionManager creates a new JPAKE session manager
func NewJPAKESessionManager(jpakeMode, pumpX2Path, pumpX2Mode, gradleCmd, javaCmd, pumpX2JarPath string, pumpState *state.PumpState) *JPAKESessionManager {
	return &JPAKESessionManager{
		authenticators: make(map[string]JPAKEAuthenticatorInterface),
		jpakeMode:      jpakeMode,
		pumpX2Path:     pumpX2Path,
		pumpX2Mode:     pumpX2Mode,
		gradleCmd:      gradleCmd,
		javaCmd:        javaCmd,
		pumpX2JarPath:  pumpX2JarPath,
		pumpState:      pumpState,
	}
}

// GetOrCreate gets or creates an authenticator for a session. round is the
// JPAKE round of the message that triggered this call (see JPAKEHandler);
// when round is 3 and no authenticator yet exists for this session, the
// client is attempting a quick-pair reconnect (Jpake3SessionKeyRequest sent
// as the very first message, skipping rounds 1a/1b/2). That can only be
// honored if we have a cached long-term key from an earlier completed
// pairing (or one seeded via -jpake-long-term-key); otherwise it returns
// ErrJPAKEQuickPairRejected so the caller can force the client back to a
// full pairing instead of spawning pumpX2's jpake-server for round 3 (which
// requires rounds 1a/1b/2 to have already run against that same process and
// will fail outright).
func (m *JPAKESessionManager) GetOrCreate(sessionID string, pairingCode string, bridge *pumpx2.Bridge, round int) (JPAKEAuthenticatorInterface, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if auth, exists := m.authenticators[sessionID]; exists {
		return auth, nil
	}

	if round == 3 {
		longTermKey := m.pumpState.GetLongTermKey()
		if len(longTermKey) == 0 {
			return nil, ErrJPAKEQuickPairRejected
		}
		log.Infof("Quick-pair reconnect detected for session %s (Jpake3SessionKeyRequest with no prior rounds); resuming from cached long-term key", sessionID)
		auth := NewQuickReconnectJPAKEAuthenticator(longTermKey)
		m.authenticators[sessionID] = auth
		return auth, nil
	}

	var auth JPAKEAuthenticatorInterface

	if m.jpakeMode == "pumpx2" {
		log.Infof("Creating pumpX2-based JPAKE authenticator for session: %s", sessionID)
		auth = NewPumpX2JPAKEAuthenticator(pairingCode, bridge, m.pumpX2Path, m.pumpX2Mode, m.gradleCmd, m.javaCmd, m.pumpX2JarPath)
	} else {
		log.Infof("Creating Go-based JPAKE authenticator for session: %s", sessionID)
		auth = NewJPAKEAuthenticator(pairingCode, bridge)
	}

	m.authenticators[sessionID] = auth
	log.Debugf("Created new JPAKE authenticator (%s mode) for session: %s", m.jpakeMode, sessionID)

	return auth, nil
}

// Remove removes an authenticator for a session
func (m *JPAKESessionManager) Remove(sessionID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.authenticators, sessionID)
	log.Debugf("Removed JPAKE authenticator for session: %s", sessionID)
}

// RemoveAll clears every in-progress authenticator. Called on BLE disconnect
// so a stale/broken authenticator (e.g. one whose pumpX2 subprocess died
// mid-handshake) is never reused by the next connection attempt.
func (m *JPAKESessionManager) RemoveAll() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(m.authenticators) == 0 {
		return
	}
	m.authenticators = make(map[string]JPAKEAuthenticatorInterface)
	log.Debug("Cleared all in-progress JPAKE authenticators")
}

// JPAKEHandler handles JPAKE authentication messages
// JPAKE is a password-authenticated key exchange protocol
type JPAKEHandler struct {
	bridge         *pumpx2.Bridge
	messageType    string
	round          int
	sessionManager *JPAKESessionManager
}

// NewJPAKEHandler creates a new JPAKE handler for a specific round
func NewJPAKEHandler(bridge *pumpx2.Bridge, sessionManager *JPAKESessionManager, messageType string, round int) *JPAKEHandler {
	return &JPAKEHandler{
		bridge:         bridge,
		sessionManager: sessionManager,
		messageType:    messageType,
		round:          round,
	}
}

// MessageType returns the message type this handler processes
func (h *JPAKEHandler) MessageType() string {
	return h.messageType
}

// RequiresAuth returns true if this message requires authentication
func (h *JPAKEHandler) RequiresAuth() bool {
	return false // JPAKE is part of the authentication process
}

// HandleMessage processes a JPAKE message
func (h *JPAKEHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling %s (round %d): txID=%d", h.messageType, h.round, msg.TxID)

	// Get or create JPAKE authenticator for this session
	// Using a simplified session ID for now (could be based on BLE connection)
	sessionID := "default"
	pairingCode := pumpState.GetPairingCode()

	auth, err := h.sessionManager.GetOrCreate(sessionID, pairingCode, h.bridge, h.round)
	if err != nil {
		return nil, err
	}

	// PumpX2JPAKEAuthenticator.encodeClientRequest needs the message name to
	// re-encode the client's request for pumpX2's real jpake-server subprocess;
	// the Go-based authenticator ignores unrecognized keys, so this is harmless
	// there. rawPacketsHex lets it forward the client's original fragments to
	// jpake-server verbatim instead of re-encoding them through a second,
	// costly cliparser JVM invocation (jpake-server doesn't validate txID, so
	// the original bytes work as-is).
	requestData := make(map[string]interface{}, len(msg.Cargo)+2)
	for k, v := range msg.Cargo {
		requestData[k] = v
	}
	requestData["messageName"] = h.messageType
	requestData["rawPacketsHex"] = msg.RawPacketsHex

	// Process this round
	responseParams, err := auth.ProcessRound(h.round, requestData)
	if err != nil {
		return nil, fmt.Errorf("JPAKE round %d failed: %w", h.round, err)
	}

	log.Debugf("JPAKE round %d processed successfully", h.round)

	// Determine the response message type
	responseType := h.getResponseType()

	// Build response using pumpX2 bridge
	response, err := h.bridge.EncodeMessage(msg.TxID, responseType, responseParams)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", responseType, err)
	}

	// Check if this is the final round (authentication complete)
	stateChanges := []StateChange{}
	if h.isFinalRound() && auth.IsComplete() {
		log.Info("JPAKE authentication complete!")

		// Get the shared secret
		sharedSecret, err := auth.GetSharedSecret()
		if err != nil {
			log.Errorf("Failed to get shared secret: %v", err)
			sharedSecret = []byte("jpake_fallback_key")
		}

		// Mark as authenticated
		stateChanges = append(stateChanges, StateChange{
			Type: StateChangeAuth,
			Data: sharedSecret,
		})

		// Cache the long-term key so a later BLE reconnect that quick-pairs
		// (Jpake3SessionKeyRequest sent directly) can be honored without a
		// full renegotiation. This is a no-op re-set when auth is itself a
		// QuickReconnectJPAKEAuthenticator, since it just returns the same
		// cached secret it was resumed from.
		if longTermSecret, err := auth.GetLongTermSecret(); err != nil {
			log.Warnf("Failed to get long-term key for caching: %v", err)
		} else if len(longTermSecret) > 0 {
			pumpState.SetLongTermKey(longTermSecret)
		}

		// Clean up the authenticator
		h.sessionManager.Remove(sessionID)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
		StateChanges:    stateChanges,
	}, nil
}

// getResponseType returns the response message type for this request
func (h *JPAKEHandler) getResponseType() string {
	// Map request to response using the real protocol's message names
	switch h.messageType {
	case "Jpake1aRequest":
		return "Jpake1aResponse"
	case "Jpake1bRequest":
		return "Jpake1bResponse"
	case "Jpake2Request":
		return "Jpake2Response"
	case "Jpake3SessionKeyRequest":
		return "Jpake3SessionKeyResponse"
	case "Jpake4KeyConfirmationRequest":
		return "Jpake4KeyConfirmationResponse"
	default:
		return h.messageType + "Response"
	}
}

// isFinalRound returns true if this is the final JPAKE round
func (h *JPAKEHandler) isFinalRound() bool {
	return h.round == 4
}

// PumpChallengeHandler handles legacy pump challenge authentication
type PumpChallengeHandler struct {
	bridge *pumpx2.Bridge
}

// NewPumpChallengeHandler creates a new pump challenge handler
func NewPumpChallengeHandler(bridge *pumpx2.Bridge) *PumpChallengeHandler {
	return &PumpChallengeHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *PumpChallengeHandler) MessageType() string {
	return "PumpChallengeRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *PumpChallengeHandler) RequiresAuth() bool {
	return false // This is part of authentication
}

// HandleMessage processes a PumpChallengeRequest (legacy authentication)
func (h *PumpChallengeHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling PumpChallengeRequest (legacy auth): txID=%d", msg.TxID)

	// For legacy auth, the client provides HMAC computed from pairing code
	// We need to verify it matches our expectation

	// Build response
	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"PumpChallengeResponse",
		map[string]interface{}{
			// Response parameters would go here
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode PumpChallengeResponse: %w", err)
	}

	// Mark as authenticated
	log.Info("Legacy authentication complete!")
	authKey := []byte("legacy_auth_key")

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
		StateChanges: []StateChange{
			{
				Type: StateChangeAuth,
				Data: authKey,
			},
		},
	}, nil
}
