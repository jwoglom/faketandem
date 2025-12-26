package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// ApiVersionHandler handles ApiVersionRequest messages
type ApiVersionHandler struct {
	bridge *pumpx2.Bridge
}

// NewApiVersionHandler creates a new API version handler
func NewApiVersionHandler(bridge *pumpx2.Bridge) *ApiVersionHandler {
	return &ApiVersionHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *ApiVersionHandler) MessageType() string {
	return "ApiVersionRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *ApiVersionHandler) RequiresAuth() bool {
	return false // ApiVersion doesn't require authentication
}

// HandleMessage processes an ApiVersionRequest
func (h *ApiVersionHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*HandlerResponse, error) {
	log.Infof("Handling ApiVersionRequest: txID=%d", msg.TxID)

	// Get the API version from pump state
	apiVersion := pumpState.GetAPIVersion()

	log.Debugf("Responding with API version: %d", apiVersion)

	// Build response using pumpX2 bridge
	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"ApiVersionResponse",
		map[string]interface{}{
			"v1": apiVersion,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode ApiVersionResponse: %w", err)
	}

	return &HandlerResponse{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}
