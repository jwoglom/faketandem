package handler

import (
	"fmt"

	"github.com/avereha/pod/pkg/pumpx2"
	"github.com/avereha/pod/pkg/state"

	log "github.com/sirupsen/logrus"
)

// TimeSinceResetHandler handles TimeSinceResetRequest messages
type TimeSinceResetHandler struct {
	bridge *pumpx2.Bridge
}

// NewTimeSinceResetHandler creates a new time since reset handler
func NewTimeSinceResetHandler(bridge *pumpx2.Bridge) *TimeSinceResetHandler {
	return &TimeSinceResetHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *TimeSinceResetHandler) MessageType() string {
	return "TimeSinceResetRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *TimeSinceResetHandler) RequiresAuth() bool {
	return false // TimeSinceReset can be sent before authentication
}

// HandleMessage processes a TimeSinceResetRequest
func (h *TimeSinceResetHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*HandlerResponse, error) {
	log.Infof("Handling TimeSinceResetRequest: txID=%d", msg.TxID)

	// Update the time since reset
	pumpState.UpdateTimeSinceReset()
	timeSinceReset := pumpState.GetTimeSinceReset()

	log.Debugf("Responding with time since reset: %d seconds", timeSinceReset)

	// Update bridge with current time since reset
	h.bridge.SetTimeSinceReset(timeSinceReset)

	// Build response using pumpX2 bridge
	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"TimeSinceResetResponse",
		map[string]interface{}{
			"timeSinceReset": timeSinceReset,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode TimeSinceResetResponse: %w", err)
	}

	return &HandlerResponse{
		ResponseMessage: response,
		Immediate:       true,
		StateChanges: []StateChange{
			{
				Type: StateChangeTime,
			},
		},
	}, nil
}
