package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

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
func (h *HistoryLogHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
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

	// Get entries from pump state
	entries := pumpState.GetHistoryLogEntries(startSeq, endSeq)

	// Build entry data for response
	entryData := make([]interface{}, 0, len(entries))
	for _, entry := range entries {
		entryMap := map[string]interface{}{
			"sequence":  entry.Sequence,
			"type":      entry.Type,
			"timestamp": entry.Timestamp.Unix(),
		}
		for k, v := range entry.Data {
			entryMap[k] = v
		}
		entryData = append(entryData, entryMap)
	}

	actualEndSeq := startSeq
	if len(entries) > 0 {
		actualEndSeq = entries[len(entries)-1].Sequence
	}

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"HistoryLogResponse",
		map[string]interface{}{
			"startSequence": startSeq,
			"endSequence":   actualEndSeq,
			"entries":       entryData,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode HistoryLogResponse: %w", err)
	}

	log.Debugf("Sent history log response with %d entries", len(entries))

	return &Response{
		ResponseMessage: response,
		Characteristic:  bluetooth.CharHistoryLog,
		Immediate:       true,
	}, nil
}

// CreateHistoryLogHandler handles CreateHistoryLogRequest messages
type CreateHistoryLogHandler struct {
	bridge *pumpx2.Bridge
}

// NewCreateHistoryLogHandler creates a new create history log handler
func NewCreateHistoryLogHandler(bridge *pumpx2.Bridge) *CreateHistoryLogHandler {
	return &CreateHistoryLogHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *CreateHistoryLogHandler) MessageType() string {
	return "CreateHistoryLogRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *CreateHistoryLogHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a CreateHistoryLogRequest
func (h *CreateHistoryLogHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling CreateHistoryLogRequest: txID=%d", msg.TxID)

	numberOfLogs := uint32(0)
	if val, ok := msg.Cargo["numberOfLogs"].(float64); ok {
		numberOfLogs = uint32(val)
	}

	log.Debugf("Create history log requested: numberOfLogs=%d", numberOfLogs)

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"CreateHistoryLogResponse",
		map[string]interface{}{
			"status": 0, // 0 = success
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode CreateHistoryLogResponse: %w", err)
	}

	return &Response{
		ResponseMessage: response,
		Characteristic:  bluetooth.CharHistoryLog,
		Immediate:       true,
	}, nil
}

// HistoryLogStatusHandler handles HistoryLogStatusRequest messages
type HistoryLogStatusHandler struct {
	bridge *pumpx2.Bridge
}

// NewHistoryLogStatusHandler creates a new history log status handler
func NewHistoryLogStatusHandler(bridge *pumpx2.Bridge) *HistoryLogStatusHandler {
	return &HistoryLogStatusHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *HistoryLogStatusHandler) MessageType() string {
	return "HistoryLogStatusRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *HistoryLogStatusHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a HistoryLogStatusRequest
func (h *HistoryLogStatusHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling HistoryLogStatusRequest: txID=%d", msg.TxID)

	numEntries := pumpState.GetHistoryLogCount()

	log.Debugf("History log status: numEntries=%d", numEntries)

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"HistoryLogStatusResponse",
		map[string]interface{}{
			"numEntries":    numEntries,
			"firstSequence": 0,
			"lastSequence":  numEntries,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode HistoryLogStatusResponse: %w", err)
	}

	return &Response{
		ResponseMessage: response,
		Characteristic:  bluetooth.CharCurrentStatus,
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
func (h *DefaultHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
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

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}
