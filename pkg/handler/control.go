package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// StreamDataReadinessHandler handles StreamDataReadinessRequest messages
type StreamDataReadinessHandler struct {
	bridge *pumpx2.Bridge
}

// NewStreamDataReadinessHandler creates a new stream data readiness handler
func NewStreamDataReadinessHandler(bridge *pumpx2.Bridge) *StreamDataReadinessHandler {
	return &StreamDataReadinessHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *StreamDataReadinessHandler) MessageType() string {
	return "StreamDataReadinessRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *StreamDataReadinessHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a StreamDataReadinessRequest
func (h *StreamDataReadinessHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling StreamDataReadinessRequest: txID=%d", msg.TxID)

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"StreamDataReadinessResponse",
		map[string]interface{}{
			"freestyleLibre2ReadinessId": 0,
			"streamDataTypeOrdinal":      0,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode StreamDataReadinessResponse: %w", err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// FactoryResetBHandler handles FactoryResetBRequest messages
type FactoryResetBHandler struct {
	bridge *pumpx2.Bridge
}

// NewFactoryResetBHandler creates a new factory reset B handler
func NewFactoryResetBHandler(bridge *pumpx2.Bridge) *FactoryResetBHandler {
	return &FactoryResetBHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *FactoryResetBHandler) MessageType() string {
	return "FactoryResetBRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *FactoryResetBHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a FactoryResetBRequest
func (h *FactoryResetBHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling FactoryResetBRequest: txID=%d", msg.TxID)

	log.Warn("Factory reset B requested — this is a simulated pump, ignoring actual reset")

	// FactoryResetBResponse has no fields at all (size=0); only a no-arg constructor exists.
	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"FactoryResetBResponse",
		map[string]interface{}{},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode FactoryResetBResponse: %w", err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}
