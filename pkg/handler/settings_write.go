package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/settings"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// SettingsWriteHandler handles settings write requests by updating the settings
// manager so subsequent reads reflect the new values, and returns success.
type SettingsWriteHandler struct {
	bridge          *pumpx2.Bridge
	settingsManager *settings.Manager
	msgType         string
	responseType    string
	readbackKey     string // which GenericSettings key to update on write
}

// NewSettingsWriteHandler creates a settings write handler
func NewSettingsWriteHandler(bridge *pumpx2.Bridge, sm *settings.Manager, msgType, readbackKey string) *SettingsWriteHandler {
	responseType := msgType[:len(msgType)-7] + "Response"
	return &SettingsWriteHandler{
		bridge:          bridge,
		settingsManager: sm,
		msgType:         msgType,
		responseType:    responseType,
		readbackKey:     readbackKey,
	}
}

// MessageType returns the message type
func (h *SettingsWriteHandler) MessageType() string { return h.msgType }

// RequiresAuth returns true
func (h *SettingsWriteHandler) RequiresAuth() bool { return true }

// HandleMessage processes a settings write request
func (h *SettingsWriteHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling %s: txID=%d cargo=%v", h.msgType, msg.TxID, msg.Cargo)

	// Update the readback key in settings manager so subsequent reads reflect the write
	if h.readbackKey != "" && msg.Cargo != nil {
		if err := h.settingsManager.UpdateConstant(h.readbackKey, msg.Cargo); err != nil {
			log.Warnf("Failed to update settings readback for %s: %v", h.readbackKey, err)
		}
	}

	response, err := h.bridge.EncodeMessage(msg.TxID, h.responseType, map[string]interface{}{
		"status": 0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", h.responseType, err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// SetModesHandler handles SetModesRequest
type SetModesHandler struct {
	bridge *pumpx2.Bridge
}

// NewSetModesHandler creates a new SetModes handler
func NewSetModesHandler(bridge *pumpx2.Bridge) *SetModesHandler {
	return &SetModesHandler{bridge: bridge}
}

// MessageType returns the message type
func (h *SetModesHandler) MessageType() string { return "SetModesRequest" }

// RequiresAuth returns true
func (h *SetModesHandler) RequiresAuth() bool { return true }

// HandleMessage processes a SetModesRequest
func (h *SetModesHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling SetModesRequest: txID=%d cargo=%v", msg.TxID, msg.Cargo)

	if mode, ok := msg.Cargo["mode"].(float64); ok {
		pumpState.SetControlIQMode(int(mode))
	}

	response, err := h.bridge.EncodeMessage(msg.TxID, "SetModesResponse", map[string]interface{}{
		"status": 0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode SetModesResponse: %w", err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}
