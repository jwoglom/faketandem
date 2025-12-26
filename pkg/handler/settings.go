package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// BasalIQSettingsHandler handles BasalIQSettingsRequest messages
type BasalIQSettingsHandler struct {
	bridge *pumpx2.Bridge
}

// NewBasalIQSettingsHandler creates a new basal IQ settings handler
func NewBasalIQSettingsHandler(bridge *pumpx2.Bridge) *BasalIQSettingsHandler {
	return &BasalIQSettingsHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *BasalIQSettingsHandler) MessageType() string {
	return "BasalIQSettingsRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *BasalIQSettingsHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a BasalIQSettingsRequest
func (h *BasalIQSettingsHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*HandlerResponse, error) {
	log.Infof("Handling BasalIQSettingsRequest: txID=%d", msg.TxID)

	// Return basal IQ settings (placeholders)
	settings := map[string]interface{}{
		"enabled":    true,
		"targetBG":   110.0, // mg/dL
		"correction": true,
	}

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"BasalIQSettingsResponse",
		settings,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode BasalIQSettingsResponse: %w", err)
	}

	return &HandlerResponse{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// ControlIQSettingsHandler handles ControlIQSettingsRequest messages
type ControlIQSettingsHandler struct {
	bridge *pumpx2.Bridge
}

// NewControlIQSettingsHandler creates a new Control-IQ settings handler
func NewControlIQSettingsHandler(bridge *pumpx2.Bridge) *ControlIQSettingsHandler {
	return &ControlIQSettingsHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *ControlIQSettingsHandler) MessageType() string {
	return "ControlIQSettingsRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *ControlIQSettingsHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a ControlIQSettingsRequest
func (h *ControlIQSettingsHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*HandlerResponse, error) {
	log.Infof("Handling ControlIQSettingsRequest: txID=%d", msg.TxID)

	// Return Control-IQ settings (placeholders)
	settings := map[string]interface{}{
		"enabled":      true,
		"sleepMode":    false,
		"exerciseMode": false,
	}

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"ControlIQSettingsResponse",
		settings,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode ControlIQSettingsResponse: %w", err)
	}

	return &HandlerResponse{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// ProfileBasalHandler handles ProfileBasalRequest messages
type ProfileBasalHandler struct {
	bridge *pumpx2.Bridge
}

// NewProfileBasalHandler creates a new profile basal handler
func NewProfileBasalHandler(bridge *pumpx2.Bridge) *ProfileBasalHandler {
	return &ProfileBasalHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *ProfileBasalHandler) MessageType() string {
	return "ProfileBasalRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *ProfileBasalHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a ProfileBasalRequest
func (h *ProfileBasalHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*HandlerResponse, error) {
	log.Infof("Handling ProfileBasalRequest: txID=%d", msg.TxID)

	// Return basal profile (simplified - single rate)
	profile := map[string]interface{}{
		"basalRate": pumpState.GetBasalRate(),
		"segments":  1, // Single segment for simplicity
	}

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"ProfileBasalResponse",
		profile,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode ProfileBasalResponse: %w", err)
	}

	return &HandlerResponse{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// GlobalsHandler handles various "Globals" request messages
type GlobalsHandler struct {
	bridge      *pumpx2.Bridge
	messageType string
}

// NewGlobalsHandler creates a new globals handler
func NewGlobalsHandler(bridge *pumpx2.Bridge, messageType string) *GlobalsHandler {
	return &GlobalsHandler{
		bridge:      bridge,
		messageType: messageType,
	}
}

// MessageType returns the message type this handler processes
func (h *GlobalsHandler) MessageType() string {
	return h.messageType
}

// RequiresAuth returns true if this message requires authentication
func (h *GlobalsHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a globals request
func (h *GlobalsHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*HandlerResponse, error) {
	log.Infof("Handling %s: txID=%d", h.messageType, msg.TxID)

	// Determine response type
	responseType := h.messageType
	if len(responseType) > 7 && responseType[len(responseType)-7:] == "Request" {
		responseType = responseType[:len(responseType)-7] + "Response"
	}

	// Build generic response with pump globals
	globals := map[string]interface{}{
		"maxBasalRate": 5.0,  // U/hr
		"maxBolus":     25.0, // U
		"insulinType":  "Humalog",
	}

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		responseType,
		globals,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", responseType, err)
	}

	return &HandlerResponse{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}
