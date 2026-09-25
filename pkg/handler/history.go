package handler

import (
	"fmt"
	"sync/atomic"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// HistoryLogHandler handles HistoryLogRequest messages: it acknowledges the
// request and then streams the requested records back as
// HistoryLogStreamResponse notifications, the way a real pump does.
type HistoryLogHandler struct {
	bridge *pumpx2.Bridge

	// nextStreamID allocates the stream id echoed in the ack and in every
	// record message of that stream. The driver does not key off it, but a real
	// pump varies it per request and tests read it back.
	nextStreamID atomic.Uint32
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

// HandleMessage processes a HistoryLogRequest.
//
// pumpX2's HistoryLogRequest carries startLog (uint32) and numberOfLogs (uint8),
// and the pump streams numberOfLogs records *backwards* from startLog -- so a
// request is for the closed sequence range [startLog-numberOfLogs+1, startLog].
// TandemKit's HistoryLogFetcher relies on exactly that: it walks a window
// backward in chunks and issues each chunk with the chunk's highest sequence
// number (HistoryLogFetcher.walkRange).
//
// The reply is a two-part exchange:
//
//   - HistoryLogResponse(status, streamId) goes back on CURRENT_STATUS, the
//     characteristic the request arrived on. It used to be forced onto
//     HISTORY_LOG, where the driver funnels everything into its stream
//     accumulator, so the ack was never delivered as an ack at all.
//   - One HistoryLogStreamResponse per record then goes out on HISTORY_LOG,
//     newest sequence first. They are built by the native encoder because
//     cliparser cannot encode this message.
func (h *HistoryLogHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling HistoryLogRequest: txID=%d", msg.TxID)

	startLog := uint32(0)
	if val, ok := cargoInt(msg, "startLog", "startSequence"); ok && val >= 0 {
		startLog = uint32(val)
	}
	numberOfLogs := uint32(1)
	if val, ok := cargoInt(msg, "numberOfLogs"); ok && val > 0 {
		numberOfLogs = uint32(val)
	}
	if numberOfLogs > 255 {
		numberOfLogs = 255
	}

	// The range walks backward from startLog and stops at sequence 1, since
	// sequence numbers are allocated from 1 up.
	endSeq := startLog
	startSeq := uint32(1)
	if startLog >= numberOfLogs {
		startSeq = startLog - numberOfLogs + 1
	}

	streamID := uint8(h.nextStreamID.Add(1))
	if streamID == 0 {
		streamID = uint8(h.nextStreamID.Add(1))
	}

	entries := pumpState.GetHistoryLogEntries(startSeq, endSeq)
	log.Debugf("History log requested: startLog=%d, numberOfLogs=%d (sequences %d..%d), %d stored, streamId=%d",
		startLog, numberOfLogs, startSeq, endSeq, len(entries), streamID)

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"HistoryLogResponse",
		map[string]interface{}{
			"status":   0,
			"streamId": int(streamID),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encode HistoryLogResponse: %w", err)
	}

	stream, err := buildHistoryLogStream(uint8(msg.TxID), streamID, entries)
	if err != nil {
		return nil, err
	}

	return &Response{
		ResponseMessage:     response,
		Characteristic:      bluetooth.CharCurrentStatus,
		NativeNotifications: stream,
		Immediate:           true,
	}, nil
}

// buildHistoryLogStream encodes one HistoryLogStreamResponse per entry, newest
// sequence first.
//
// One record per message is what a real pump sends: pumpX2 declares
// HistoryLogStreamResponse's size as 28, which is exactly the two-byte
// [count][streamId] prefix plus a single 26-byte record. The driver
// deduplicates by sequence number, so the ordering here matters only for
// realism.
func buildHistoryLogStream(txID, streamID uint8, entries []state.HistoryLogEntry) ([]*protocol.NativeMessage, error) {
	messages := make([]*protocol.NativeMessage, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- {
		record := entries[i].EncodeRecord()
		msg, err := protocol.BuildHistoryLogStreamResponse(txID, streamID, [][]byte{record})
		if err != nil {
			return nil, fmt.Errorf("failed to build HistoryLogStreamResponse for sequence %d: %w",
				entries[i].Sequence, err)
		}
		messages = append(messages, msg)
	}
	return messages, nil
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
