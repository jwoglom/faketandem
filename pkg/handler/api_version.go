package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// APIVersionHandler handles ApiVersionRequest messages
type APIVersionHandler struct {
	bridge *pumpx2.Bridge
}

// NewAPIVersionHandler creates a new API version handler
func NewAPIVersionHandler(bridge *pumpx2.Bridge) *APIVersionHandler {
	return &APIVersionHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *APIVersionHandler) MessageType() string {
	return "ApiVersionRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *APIVersionHandler) RequiresAuth() bool {
	return false // ApiVersion doesn't require authentication
}

// HandleMessage processes an ApiVersionRequest
func (h *APIVersionHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling ApiVersionRequest: txID=%d", msg.TxID)

	// Get the API version from pump state (major.minor format)
	major := pumpState.GetAPIVersionMajor()
	minor := pumpState.GetAPIVersionMinor()

	log.Debugf("Responding with API version: %d.%d", major, minor)

	// Build response using pumpX2 bridge
	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"ApiVersionResponse",
		map[string]interface{}{
			"v1": major,
			"v2": minor,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode ApiVersionResponse: %w", err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}
