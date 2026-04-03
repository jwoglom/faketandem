package handler

import (
	"fmt"
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// SuspendPumpingHandler handles SuspendPumpingRequest messages
type SuspendPumpingHandler struct {
	bridge *pumpx2.Bridge
}

// NewSuspendPumpingHandler creates a new suspend pumping handler
func NewSuspendPumpingHandler(bridge *pumpx2.Bridge) *SuspendPumpingHandler {
	return &SuspendPumpingHandler{bridge: bridge}
}

// MessageType returns the message type this handler processes
func (h *SuspendPumpingHandler) MessageType() string {
	return "SuspendPumpingRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *SuspendPumpingHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a SuspendPumpingRequest
func (h *SuspendPumpingHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling SuspendPumpingRequest: txID=%d", msg.TxID)

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"SuspendPumpingResponse",
		map[string]interface{}{
			"status": 0,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encode SuspendPumpingResponse: %w", err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
		StateChanges: []StateChange{
			{Type: StateChangeSuspend, Data: true},
		},
	}, nil
}

// ResumePumpingHandler handles ResumePumpingRequest messages
type ResumePumpingHandler struct {
	bridge *pumpx2.Bridge
}

// NewResumePumpingHandler creates a new resume pumping handler
func NewResumePumpingHandler(bridge *pumpx2.Bridge) *ResumePumpingHandler {
	return &ResumePumpingHandler{bridge: bridge}
}

// MessageType returns the message type this handler processes
func (h *ResumePumpingHandler) MessageType() string {
	return "ResumePumpingRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *ResumePumpingHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a ResumePumpingRequest
func (h *ResumePumpingHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling ResumePumpingRequest: txID=%d", msg.TxID)

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"ResumePumpingResponse",
		map[string]interface{}{
			"status": 0,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encode ResumePumpingResponse: %w", err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
		StateChanges: []StateChange{
			{Type: StateChangeSuspend, Data: false},
		},
	}, nil
}

// SetTempRateHandler handles SetTempRateRequest messages
type SetTempRateHandler struct {
	bridge *pumpx2.Bridge
}

// NewSetTempRateHandler creates a new set temp rate handler
func NewSetTempRateHandler(bridge *pumpx2.Bridge) *SetTempRateHandler {
	return &SetTempRateHandler{bridge: bridge}
}

// MessageType returns the message type this handler processes
func (h *SetTempRateHandler) MessageType() string {
	return "SetTempRateRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *SetTempRateHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a SetTempRateRequest
func (h *SetTempRateHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling SetTempRateRequest: txID=%d cargo=%v", msg.TxID, msg.Cargo)

	percentage := 100
	if val, ok := msg.Cargo["percentage"].(float64); ok {
		percentage = int(val)
	}
	durationMinutes := 0
	if val, ok := msg.Cargo["duration"].(float64); ok {
		durationMinutes = int(val)
	}

	basalRate := pumpState.GetBasalRate()
	tempRate := basalRate * float64(percentage) / 100.0
	tempEnd := time.Now().Add(time.Duration(durationMinutes) * time.Minute)

	log.Infof("Setting temp rate: %d%% (%.3f U/hr) for %d minutes", percentage, tempRate, durationMinutes)

	stateChanges := []StateChange{
		{
			Type: StateChangeBasal,
			Data: &state.BasalState{
				CurrentRate:     basalRate,
				TempBasalActive: true,
				TempBasalRate:   tempRate,
				TempBasalEnd:    tempEnd,
			},
		},
	}

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"SetTempRateResponse",
		map[string]interface{}{
			"status": 0,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encode SetTempRateResponse: %w", err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
		StateChanges:    stateChanges,
	}, nil
}

// StopTempRateHandler handles StopTempRateRequest messages
type StopTempRateHandler struct {
	bridge *pumpx2.Bridge
}

// NewStopTempRateHandler creates a new stop temp rate handler
func NewStopTempRateHandler(bridge *pumpx2.Bridge) *StopTempRateHandler {
	return &StopTempRateHandler{bridge: bridge}
}

// MessageType returns the message type this handler processes
func (h *StopTempRateHandler) MessageType() string {
	return "StopTempRateRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *StopTempRateHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a StopTempRateRequest
func (h *StopTempRateHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling StopTempRateRequest: txID=%d", msg.TxID)

	basalRate := pumpState.GetBasalRate()

	stateChanges := []StateChange{
		{
			Type: StateChangeBasal,
			Data: &state.BasalState{
				CurrentRate:     basalRate,
				TempBasalActive: false,
			},
		},
	}

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"StopTempRateResponse",
		map[string]interface{}{
			"status": 0,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encode StopTempRateResponse: %w", err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
		StateChanges:    stateChanges,
	}, nil
}

// SimpleControlHandler handles simple control requests that just return success
type SimpleControlHandler struct {
	bridge       *pumpx2.Bridge
	msgType      string
	responseType string
}

// NewSimpleControlHandler creates a new simple control handler
func NewSimpleControlHandler(bridge *pumpx2.Bridge, msgType string) *SimpleControlHandler {
	responseType := msgType
	if len(responseType) > 7 && responseType[len(responseType)-7:] == "Request" {
		responseType = responseType[:len(responseType)-7] + "Response"
	}
	return &SimpleControlHandler{
		bridge:       bridge,
		msgType:      msgType,
		responseType: responseType,
	}
}

// MessageType returns the message type this handler processes
func (h *SimpleControlHandler) MessageType() string {
	return h.msgType
}

// RequiresAuth returns true if this message requires authentication
func (h *SimpleControlHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes the request and returns a success response
func (h *SimpleControlHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling %s: txID=%d cargo=%v", h.msgType, msg.TxID, msg.Cargo)

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		h.responseType,
		map[string]interface{}{
			"status": 0,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", h.responseType, err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}
