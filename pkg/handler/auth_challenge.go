package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// CentralChallengeHandler handles CentralChallengeRequest messages
// This is the first step in the authentication flow
type CentralChallengeHandler struct {
	bridge *pumpx2.Bridge
}

// NewCentralChallengeHandler creates a new central challenge handler
func NewCentralChallengeHandler(bridge *pumpx2.Bridge) *CentralChallengeHandler {
	return &CentralChallengeHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *CentralChallengeHandler) MessageType() string {
	return "CentralChallengeRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *CentralChallengeHandler) RequiresAuth() bool {
	return false // This is part of the authentication flow itself
}

// HandleMessage processes a CentralChallengeRequest
func (h *CentralChallengeHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*HandlerResponse, error) {
	log.Infof("Handling CentralChallengeRequest: txID=%d", msg.TxID)
	log.Info("Client is initiating authentication")

	// Extract the appInstanceID from the request if present
	appInstanceID := uint32(0)
	if val, ok := msg.Cargo["appInstanceId"].(float64); ok {
		appInstanceID = uint32(val)
	} else if val, ok := msg.Cargo["appInstanceID"].(float64); ok {
		appInstanceID = uint32(val)
	}

	log.Debugf("App instance ID: %d", appInstanceID)

	// Build response using pumpX2 bridge
	// For JPAKE authentication, we need to provide parameters for the first round
	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"CentralChallengeResponse",
		map[string]interface{}{
			"appInstanceId": appInstanceID,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode CentralChallengeResponse: %w", err)
	}

	log.Info("Sent CentralChallengeResponse - waiting for JPAKE or legacy auth")

	return &HandlerResponse{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}
