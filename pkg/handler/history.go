package handler

import (
	"fmt"

	"github.com/avereha/pod/pkg/bluetooth"
	"github.com/avereha/pod/pkg/pumpx2"
	"github.com/avereha/pod/pkg/state"

	log "github.com/sirupsen/logrus"
)

// HistoryLogHandler handles HistoryLogRequest messages
type HistoryLogHandler struct {
	bridge *pumpx2.Bridge
}

// NewHistoryLogHandler creates a new history log handler
func NewHistoryLogHandler(bridge *pumpx2.Bridge) *HistoryLogHandler {
	return &HistoryLogHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *HistoryLogHandler) MessageType() string {
	return "HistoryLogRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *HistoryLogHandler) RequiresAuth() bool {
	return true // History log requires authentication
}

// HandleMessage processes a HistoryLogRequest
func (h *HistoryLogHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*HandlerResponse, error) {
	log.Infof("Handling HistoryLogRequest: txID=%d", msg.TxID)

	// Extract request parameters
	startSeq := uint32(0)
	endSeq := uint32(100)

	if val, ok := msg.Cargo["startSequence"].(float64); ok {
		startSeq = uint32(val)
	}
	if val, ok := msg.Cargo["endSequence"].(float64); ok {
		endSeq = uint32(val)
	}

	log.Debugf("History log requested: start=%d, end=%d", startSeq, endSeq)

	// For now, return an empty history log
	// TODO: Implement actual history log storage and retrieval
	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"HistoryLogResponse",
		map[string]interface{}{
			"startSequence": startSeq,
			"endSequence":   startSeq, // Empty - no entries
			"entries":       []interface{}{},
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode HistoryLogResponse: %w", err)
	}

	log.Debug("Sent empty history log response")

	return &HandlerResponse{
		ResponseMessage: response,
		Characteristic:  bluetooth.CharHistoryLog,
		Immediate:       true,
	}, nil
}

// DefaultHandler handles unknown message types
type DefaultHandler struct {
	bridge *pumpx2.Bridge
}

// NewDefaultHandler creates a new default handler
func NewDefaultHandler(bridge *pumpx2.Bridge) *DefaultHandler {
	return &DefaultHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *DefaultHandler) MessageType() string {
	return "Default"
}

// RequiresAuth returns true if this message requires authentication
func (h *DefaultHandler) RequiresAuth() bool {
	return false // We'll log unknown messages regardless of auth status
}

// HandleMessage processes an unknown message
func (h *DefaultHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*HandlerResponse, error) {
	log.Warnf("No handler for message type '%s' (opcode=%d, txID=%d)",
		msg.MessageType, msg.Opcode, msg.TxID)
	log.Debugf("Message cargo: %+v", msg.Cargo)

	// Try to build a generic response by replacing "Request" with "Response"
	// This won't always work but is better than nothing
	responseType := msg.MessageType
	if len(responseType) > 7 && responseType[len(responseType)-7:] == "Request" {
		responseType = responseType[:len(responseType)-7] + "Response"
	}

	log.Infof("Attempting to send generic response: %s", responseType)

	// Try to encode a response with minimal parameters
	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		responseType,
		map[string]interface{}{},
	)

	if err != nil {
		log.Errorf("Failed to encode generic response: %v", err)
		// Don't return error - just log it
		return nil, nil
	}

	log.Infof("Sent generic response: %s", responseType)

	return &HandlerResponse{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}
