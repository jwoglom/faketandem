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

	// pumpX2's HistoryLogRequest fields are "startLog" (uint32, the sequence
	// number to start at) and "numberOfLogs" (uint8, how many records to
	// stream) -- not "startSequence"/"endSequence". The pump streams
	// numberOfLogs records starting at startLog.
	startSeq := uint32(0)
	numberOfLogs := uint32(0)

	if val, ok := cargoInt(msg, "startLog", "startSequence"); ok && val >= 0 {
		startSeq = uint32(val)
	}
	if val, ok := cargoInt(msg, "numberOfLogs"); ok && val > 0 {
		numberOfLogs = uint32(val)
	}
	if numberOfLogs == 0 {
		numberOfLogs = 1
	}

	endSeq := startSeq + numberOfLogs - 1

	log.Debugf("History log requested: startLog=%d, numberOfLogs=%d (sequences %d..%d)",
		startSeq, numberOfLogs, startSeq, endSeq)

	// Get entries from pump state. The real HistoryLogResponse has no field
	// for embedded entries -- actual log entries go out separately via
	// HistoryLogStreamResponse messages on the history log characteristic.
	// This response just acknowledges the request and reports a stream ID.
	entries := pumpState.GetHistoryLogEntries(startSeq, endSeq)
	log.Debugf("History log entries matched: %d", len(entries))

	// HistoryLogResponse(int status, int streamId)
	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"HistoryLogResponse",
		map[string]interface{}{
			"status":   0,
			"streamId": 1,
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

	// HistoryLogStatusResponse(numEntries, firstSequenceNum, lastSequenceNum).
	// Sequence numbers must agree with what storage actually holds: entries are
	// numbered from 1, so an empty log reports 0/0 and a non-empty one reports
	// first=1, last=numEntries. Reporting firstSequence 0 for a log whose first
	// record is sequence 1 made the reported range inconsistent with every
	// HistoryLogRequest the driver would then send.
	numEntries := pumpState.GetHistoryLogCount()
	firstSequence, lastSequence := pumpState.GetHistoryLogSequenceRange()

	log.Debugf("History log status: numEntries=%d, firstSequence=%d, lastSequence=%d",
		numEntries, firstSequence, lastSequence)

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"HistoryLogStatusResponse",
		map[string]interface{}{
			"numEntries":       numEntries,
			"firstSequenceNum": firstSequence,
			"lastSequenceNum":  lastSequence,
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
