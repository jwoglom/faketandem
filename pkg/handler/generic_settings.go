package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/settings"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// GenericSettingsHandler handles configurable settings messages
type GenericSettingsHandler struct {
	bridge          *pumpx2.Bridge
	settingsManager *settings.Manager
	messageType     string
	requiresAuth    bool
}

// NewGenericSettingsHandler creates a new generic settings handler
func NewGenericSettingsHandler(bridge *pumpx2.Bridge, settingsManager *settings.Manager, messageType string, requiresAuth bool) *GenericSettingsHandler {
	return &GenericSettingsHandler{
		bridge:          bridge,
		settingsManager: settingsManager,
		messageType:     messageType,
		requiresAuth:    requiresAuth,
	}
}

// MessageType returns the message type this handler processes
func (h *GenericSettingsHandler) MessageType() string {
	return h.messageType
}

// RequiresAuth returns true if this message requires authentication
func (h *GenericSettingsHandler) RequiresAuth() bool {
	return h.requiresAuth
}

// HandleMessage processes a settings request
func (h *GenericSettingsHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling %s: txID=%d", h.messageType, msg.TxID)

	// Get response from settings manager
	responseData, err := h.settingsManager.GetResponse(h.messageType)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings response: %w", err)
	}

	log.Debugf("Settings response for %s: %v", h.messageType, responseData)

	// Determine response type (replace "Request" with "Response")
	responseType := h.messageType
	if len(responseType) > 7 && responseType[len(responseType)-7:] == "Request" {
		responseType = responseType[:len(responseType)-7] + "Response"
	}

	// Build response using pumpX2 bridge
	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		responseType,
		responseData,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", responseType, err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}
