package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// JPAKEHandler handles JPAKE authentication messages
// JPAKE is a password-authenticated key exchange protocol
type JPAKEHandler struct {
	bridge      *pumpx2.Bridge
	messageType string
	round       int
}

// NewJPAKEHandler creates a new JPAKE handler for a specific round
func NewJPAKEHandler(bridge *pumpx2.Bridge, messageType string, round int) *JPAKEHandler {
	return &JPAKEHandler{
		bridge:      bridge,
		messageType: messageType,
		round:       round,
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
func (h *JPAKEHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*HandlerResponse, error) {
	log.Infof("Handling %s (round %d): txID=%d", h.messageType, h.round, msg.TxID)

	// Get the pairing code from pump state
	pairingCode := pumpState.GetPairingCode()
	log.Debugf("Using pairing code: %s", pairingCode)

	// Use pumpX2's JPAKE functionality to handle the exchange
	// The cliparser has built-in JPAKE support
	output, err := h.bridge.ExecuteJPAKE(pairingCode)
	if err != nil {
		return nil, fmt.Errorf("JPAKE execution failed: %w", err)
	}

	log.Tracef("JPAKE output: %s", output)

	// Determine the response message type based on the request
	responseType := h.getResponseType()
	log.Debugf("Sending response: %s", responseType)

	// Build response using the JPAKE output
	// The exact parameters will depend on the JPAKE round
	response, err := h.buildJPAKEResponse(msg.TxID, responseType, msg.Cargo)
	if err != nil {
		return nil, fmt.Errorf("failed to build JPAKE response: %w", err)
	}

	// Check if this is the final round (authentication complete)
	stateChanges := []StateChange{}
	if h.isFinalRound() {
		log.Info("JPAKE authentication complete!")
		// TODO: Extract the derived key and mark as authenticated
		// For now, we'll mark as authenticated with a placeholder key
		authKey := []byte("placeholder_auth_key")
		stateChanges = append(stateChanges, StateChange{
			Type: StateChangeAuth,
			Data: authKey,
		})
	}

	return &HandlerResponse{
		ResponseMessage: response,
		Immediate:       true,
		StateChanges:    stateChanges,
	}, nil
}

// getResponseType returns the response message type for this request
func (h *JPAKEHandler) getResponseType() string {
	// Map request to response
	// The exact message types would come from pumpX2
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

// buildJPAKEResponse builds a JPAKE response message
func (h *JPAKEHandler) buildJPAKEResponse(txID int, responseType string, requestCargo map[string]interface{}) (*pumpx2.EncodedMessage, error) {
	// For now, use pumpX2 bridge to encode the response
	// The parameters would typically come from the JPAKE calculation
	params := make(map[string]interface{})

	// Copy relevant parameters from request
	// The exact parameters depend on the JPAKE round
	for key, val := range requestCargo {
		params[key] = val
	}

	response, err := h.bridge.EncodeMessage(txID, responseType, params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", responseType, err)
	}

	return response, nil
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
func (h *PumpChallengeHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*HandlerResponse, error) {
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

	return &HandlerResponse{
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
