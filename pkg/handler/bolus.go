package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// BolusPermissionHandler handles BolusPermissionRequest messages
type BolusPermissionHandler struct {
	bridge *pumpx2.Bridge
}

// NewBolusPermissionHandler creates a new bolus permission handler
func NewBolusPermissionHandler(bridge *pumpx2.Bridge) *BolusPermissionHandler {
	return &BolusPermissionHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *BolusPermissionHandler) MessageType() string {
	return "BolusPermissionRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *BolusPermissionHandler) RequiresAuth() bool {
	return true // Bolus requires authentication
}

// HandleMessage processes a BolusPermissionRequest
func (h *BolusPermissionHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*HandlerResponse, error) {
	log.Infof("Handling BolusPermissionRequest: txID=%d", msg.TxID)

	// Check if pump is in a state where bolus is allowed
	if pumpState.Bolus.Active {
		log.Warn("Bolus permission denied: bolus already active")
		// TODO: Return proper denial response
	}

	// Grant bolus permission
	log.Info("Granting bolus permission")

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"BolusPermissionResponse",
		map[string]interface{}{
			"granted": true,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode BolusPermissionResponse: %w", err)
	}

	return &HandlerResponse{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// BolusCalcDataSnapshotHandler handles BolusCalcDataSnapshotRequest messages
type BolusCalcDataSnapshotHandler struct {
	bridge *pumpx2.Bridge
}

// NewBolusCalcDataSnapshotHandler creates a new bolus calc data snapshot handler
func NewBolusCalcDataSnapshotHandler(bridge *pumpx2.Bridge) *BolusCalcDataSnapshotHandler {
	return &BolusCalcDataSnapshotHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *BolusCalcDataSnapshotHandler) MessageType() string {
	return "BolusCalcDataSnapshotRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *BolusCalcDataSnapshotHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a BolusCalcDataSnapshotRequest
func (h *BolusCalcDataSnapshotHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*HandlerResponse, error) {
	log.Infof("Handling BolusCalcDataSnapshotRequest: txID=%d", msg.TxID)

	// Provide current bolus calculation data
	calcData := map[string]interface{}{
		"iob":              pumpState.IOB,
		"basalRate":        pumpState.GetBasalRate(),
		"timeSinceReset":   pumpState.GetTimeSinceReset(),
		"bolusId":          pumpState.GetNextBolusID(),
		"carbRatio":        12.0,  // g/U - placeholder
		"correctionFactor": 50.0,  // mg/dL/U - placeholder
		"targetBG":         100.0, // mg/dL - placeholder
	}

	log.Debugf("Bolus calc data: IOB=%.2f, basal=%.2f, bolusID=%d",
		pumpState.IOB, pumpState.GetBasalRate(), pumpState.GetNextBolusID())

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"BolusCalcDataSnapshotResponse",
		calcData,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode BolusCalcDataSnapshotResponse: %w", err)
	}

	return &HandlerResponse{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// InitiateBolusHandler handles InitiateBolusRequest messages
type InitiateBolusHandler struct {
	bridge *pumpx2.Bridge
}

// NewInitiateBolusHandler creates a new initiate bolus handler
func NewInitiateBolusHandler(bridge *pumpx2.Bridge) *InitiateBolusHandler {
	return &InitiateBolusHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *InitiateBolusHandler) MessageType() string {
	return "InitiateBolusRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *InitiateBolusHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes an InitiateBolusRequest
func (h *InitiateBolusHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*HandlerResponse, error) {
	log.Infof("Handling InitiateBolusRequest: txID=%d", msg.TxID)

	// Extract bolus parameters
	bolusUnits := 0.0
	bolusID := uint32(0)

	if val, ok := msg.Cargo["insulin"].(float64); ok {
		bolusUnits = val
	} else if val, ok := msg.Cargo["units"].(float64); ok {
		bolusUnits = val
	}

	if val, ok := msg.Cargo["bolusId"].(float64); ok {
		bolusID = uint32(val)
	} else if val, ok := msg.Cargo["bolusID"].(float64); ok {
		bolusID = uint32(val)
	}

	if bolusUnits <= 0 {
		return nil, fmt.Errorf("invalid bolus units: %.2f", bolusUnits)
	}

	log.Infof("Initiating bolus: %.2f units, bolusID=%d", bolusUnits, bolusID)

	// Start the bolus
	stateChanges := []StateChange{
		{
			Type: StateChangeBolus,
			Data: &state.BolusState{
				Active:         true,
				UnitsDelivered: 0,
				UnitsTotal:     bolusUnits,
				BolusID:        bolusID,
			},
		},
	}

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"InitiateBolusResponse",
		map[string]interface{}{
			"bolusId": bolusID,
			"success": true,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode InitiateBolusResponse: %w", err)
	}

	return &HandlerResponse{
		ResponseMessage: response,
		Immediate:       true,
		StateChanges:    stateChanges,
	}, nil
}

// BolusTerminationHandler handles BolusTerminationRequest messages
type BolusTerminationHandler struct {
	bridge *pumpx2.Bridge
}

// NewBolusTerminationHandler creates a new bolus termination handler
func NewBolusTerminationHandler(bridge *pumpx2.Bridge) *BolusTerminationHandler {
	return &BolusTerminationHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *BolusTerminationHandler) MessageType() string {
	return "BolusTerminationRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *BolusTerminationHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a BolusTerminationRequest
func (h *BolusTerminationHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*HandlerResponse, error) {
	log.Infof("Handling BolusTerminationRequest: txID=%d", msg.TxID)

	if !pumpState.Bolus.Active {
		log.Warn("No active bolus to terminate")
	}

	log.Infof("Terminating bolus: delivered %.2f of %.2f units",
		pumpState.Bolus.UnitsDelivered, pumpState.Bolus.UnitsTotal)

	// Stop the bolus
	stateChanges := []StateChange{
		{
			Type: StateChangeBolus,
			Data: &state.BolusState{
				Active: false,
			},
		},
	}

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"BolusTerminationResponse",
		map[string]interface{}{
			"success": true,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode BolusTerminationResponse: %w", err)
	}

	return &HandlerResponse{
		ResponseMessage: response,
		Immediate:       true,
		StateChanges:    stateChanges,
	}, nil
}

// RemoteBgEntryHandler handles RemoteBgEntryRequest messages
type RemoteBgEntryHandler struct {
	bridge *pumpx2.Bridge
}

// NewRemoteBgEntryHandler creates a new remote BG entry handler
func NewRemoteBgEntryHandler(bridge *pumpx2.Bridge) *RemoteBgEntryHandler {
	return &RemoteBgEntryHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *RemoteBgEntryHandler) MessageType() string {
	return "RemoteBgEntryRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *RemoteBgEntryHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a RemoteBgEntryRequest
func (h *RemoteBgEntryHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*HandlerResponse, error) {
	log.Infof("Handling RemoteBgEntryRequest: txID=%d", msg.TxID)

	bgValue := 0.0
	if val, ok := msg.Cargo["bgValue"].(float64); ok {
		bgValue = val
	} else if val, ok := msg.Cargo["bg"].(float64); ok {
		bgValue = val
	}

	log.Infof("Remote BG entry: %.0f mg/dL", bgValue)

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"RemoteBgEntryResponse",
		map[string]interface{}{
			"success": true,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode RemoteBgEntryResponse: %w", err)
	}

	return &HandlerResponse{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// RemoteCarbEntryHandler handles remote carbohydrate entry
type RemoteCarbEntryHandler struct {
	bridge *pumpx2.Bridge
}

// NewRemoteCarbEntryHandler creates a new remote carb entry handler
func NewRemoteCarbEntryHandler(bridge *pumpx2.Bridge) *RemoteCarbEntryHandler {
	return &RemoteCarbEntryHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *RemoteCarbEntryHandler) MessageType() string {
	return "RemoteCarbEntryRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *RemoteCarbEntryHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a RemoteCarbEntryRequest
func (h *RemoteCarbEntryHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*HandlerResponse, error) {
	log.Infof("Handling RemoteCarbEntryRequest: txID=%d", msg.TxID)

	carbGrams := 0.0
	if val, ok := msg.Cargo["carbs"].(float64); ok {
		carbGrams = val
	} else if val, ok := msg.Cargo["carbGrams"].(float64); ok {
		carbGrams = val
	}

	log.Infof("Remote carb entry: %.0f grams", carbGrams)

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"RemoteCarbEntryResponse",
		map[string]interface{}{
			"success": true,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode RemoteCarbEntryResponse: %w", err)
	}

	return &HandlerResponse{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// BolusPermissionReleaseHandler handles BolusPermissionReleaseRequest messages
type BolusPermissionReleaseHandler struct {
	bridge *pumpx2.Bridge
}

// NewBolusPermissionReleaseHandler creates a new bolus permission release handler
func NewBolusPermissionReleaseHandler(bridge *pumpx2.Bridge) *BolusPermissionReleaseHandler {
	return &BolusPermissionReleaseHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *BolusPermissionReleaseHandler) MessageType() string {
	return "BolusPermissionReleaseRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *BolusPermissionReleaseHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a BolusPermissionReleaseRequest
func (h *BolusPermissionReleaseHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*HandlerResponse, error) {
	log.Infof("Handling BolusPermissionReleaseRequest: txID=%d", msg.TxID)

	log.Info("Releasing bolus permission")

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"BolusPermissionReleaseResponse",
		map[string]interface{}{
			"success": true,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode BolusPermissionReleaseResponse: %w", err)
	}

	return &HandlerResponse{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}
