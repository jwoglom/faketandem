package handler

import (
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
	IsComplete() bool
}

// JPAKESessionManager manages JPAKE authentication sessions
type JPAKESessionManager struct {
	authenticators map[string]JPAKEAuthenticatorInterface
	mutex          sync.RWMutex

	// Configuration for creating authenticators
	jpakeMode  string
	pumpX2Path string
	pumpX2Mode string
	gradleCmd  string
	javaCmd    string
}

// NewJPAKESessionManager creates a new JPAKE session manager
func NewJPAKESessionManager(jpakeMode, pumpX2Path, pumpX2Mode, gradleCmd, javaCmd string) *JPAKESessionManager {
	return &JPAKESessionManager{
		authenticators: make(map[string]JPAKEAuthenticatorInterface),
		jpakeMode:      jpakeMode,
		pumpX2Path:     pumpX2Path,
		pumpX2Mode:     pumpX2Mode,
		gradleCmd:      gradleCmd,
		javaCmd:        javaCmd,
	}
}

// GetOrCreate gets or creates an authenticator for a session
func (m *JPAKESessionManager) GetOrCreate(sessionID string, pairingCode string, bridge *pumpx2.Bridge) JPAKEAuthenticatorInterface {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if auth, exists := m.authenticators[sessionID]; exists {
		return auth
	}

	var auth JPAKEAuthenticatorInterface

	if m.jpakeMode == "pumpx2" {
		log.Infof("Creating pumpX2-based JPAKE authenticator for session: %s", sessionID)
		auth = NewPumpX2JPAKEAuthenticator(pairingCode, bridge, m.pumpX2Path, m.pumpX2Mode, m.gradleCmd, m.javaCmd)
	} else {
		log.Infof("Creating Go-based JPAKE authenticator for session: %s", sessionID)
		auth = NewJPAKEAuthenticator(pairingCode, bridge)
	}

	m.authenticators[sessionID] = auth
	log.Debugf("Created new JPAKE authenticator (%s mode) for session: %s", m.jpakeMode, sessionID)

	return auth
}

// Remove removes an authenticator for a session
func (m *JPAKESessionManager) Remove(sessionID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.authenticators, sessionID)
	log.Debugf("Removed JPAKE authenticator for session: %s", sessionID)
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

	auth := h.sessionManager.GetOrCreate(sessionID, pairingCode, h.bridge)

	// Process this round
	responseParams, err := auth.ProcessRound(h.round, msg.Cargo)
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
	// Map request to response
	switch h.messageType {
	case "JPAKERound1Request":
		return "JPAKERound1Response"
	case "JPAKERound2Request":
		return "JPAKERound2Response"
	case "JPAKERound3Request":
		return "JPAKERound3Response"
	case "JPAKERound4Request":
		return "JPAKERound4Response"
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
