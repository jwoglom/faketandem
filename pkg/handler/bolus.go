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
func (h *BolusPermissionHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
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
			"status":       0,
			"bolusId":      pumpState.GetNextBolusID(),
			"nackReasonId": 0,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode BolusPermissionResponse: %w", err)
	}

	return &Response{
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
func (h *BolusCalcDataSnapshotHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling BolusCalcDataSnapshotRequest: txID=%d", msg.TxID)

	// Provide current bolus calculation data. BolusCalcDataSnapshotResponse's
	// real constructor takes 13 fields (int/long amounts scaled by 1000, per
	// pumpX2's convention elsewhere) -- see BolusCalcDataSnapshotResponse.java.
	calcData := map[string]interface{}{
		"isUnacked":                 false,
		"correctionFactor":          50,                          // mg/dL/U - placeholder
		"iob":                       int64(pumpState.IOB * 1000), // milli-units
		"cartridgeRemainingInsulin": 20000,                       // milli-units - placeholder
		"targetBg":                  100,                         // mg/dL - placeholder
		"isf":                       50,                          // mg/dL/U - placeholder
		"carbEntryEnabled":          true,
		"carbRatio":                 int64(12000), // g/U * 1000 - placeholder
		"maxBolusAmount":            25000,        // milli-units - placeholder
		"maxBolusHourlyTotal":       int64(25000), // milli-units - placeholder
		"maxBolusEventsExceeded":    false,
		"maxIobEventsExceeded":      false,
		"isAutopopAllowed":          true,
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

	return &Response{
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
func (h *InitiateBolusHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling InitiateBolusRequest: txID=%d", msg.TxID)

	// Field names below are pumpX2's own (InitiateBolusRequest.java): the
	// request carries "totalVolume" in MILLIunits and "bolusID" (capital D),
	// plus optional metadata. Reading "insulin"/"units"/"bolusId" instead --
	// as this handler used to -- always left bolusUnits at 0, failed the
	// validation below, and sent no response at all, so no bolus could ever be
	// started through faketandem.
	totalVolumeMilliunits, _ := cargoInt(msg, "totalVolume")
	bolusUnits := float64(totalVolumeMilliunits) / 1000.0

	rawBolusID, _ := cargoInt(msg, "bolusID", "bolusId")
	bolusID := uint32(rawBolusID)

	bolusTypeBitmask, _ := cargoInt(msg, "bolusTypeBitmask")
	foodVolume, _ := cargoInt(msg, "foodVolume")
	correctionVolume, _ := cargoInt(msg, "correctionVolume")
	bolusCarbs, _ := cargoInt(msg, "bolusCarbs")
	bolusBG, _ := cargoInt(msg, "bolusBG")

	if bolusUnits <= 0 {
		return nil, fmt.Errorf("invalid bolus units: %.2f (totalVolume=%d mU)", bolusUnits, totalVolumeMilliunits)
	}

	log.Infof("Initiating bolus: %.2f units (%d mU), bolusID=%d, typeBitmask=%d, food=%d mU, correction=%d mU, carbs=%dg, bg=%d",
		bolusUnits, totalVolumeMilliunits, bolusID, bolusTypeBitmask, foodVolume, correctionVolume, bolusCarbs, bolusBG)

	// Start the bolus. A bolus commanded over BLE reports
	// BolusSource.BLUETOOTH_REMOTE_BOLUS (8), not the quickBolus default (0) --
	// drivers use bolusSourceId to tell an app-commanded bolus from one
	// programmed on the pump's own UI.
	stateChanges := []StateChange{
		{
			Type: StateChangeBolus,
			Data: &state.BolusState{
				Active:           true,
				UnitsDelivered:   0,
				UnitsTotal:       bolusUnits,
				BolusID:          bolusID,
				SourceID:         state.BolusSourceBluetoothRemote,
				TypeBitmask:      int(bolusTypeBitmask),
				FoodVolume:       float64(foodVolume) / 1000.0,
				CorrectionVolume: float64(correctionVolume) / 1000.0,
			},
		},
	}

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"InitiateBolusResponse",
		map[string]interface{}{
			"status":       0,
			"bolusId":      bolusID,
			"statusTypeId": 0,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode InitiateBolusResponse: %w", err)
	}

	return &Response{
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
func (h *RemoteBgEntryHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling RemoteBgEntryRequest: txID=%d", msg.TxID)

	// pumpX2's RemoteBgEntryRequest field is "bg" (mg/dL).
	bgValue, _ := cargoFloat(msg, "bg", "bgValue")

	log.Infof("Remote BG entry: %.0f mg/dL", bgValue)

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"RemoteBgEntryResponse",
		map[string]interface{}{
			"status": 0,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode RemoteBgEntryResponse: %w", err)
	}

	return &Response{
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
func (h *RemoteCarbEntryHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling RemoteCarbEntryRequest: txID=%d", msg.TxID)

	// pumpX2's RemoteCarbEntryRequest field is "carbs" (grams).
	carbGrams, _ := cargoFloat(msg, "carbs", "carbGrams")

	log.Infof("Remote carb entry: %.0f grams", carbGrams)

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"RemoteCarbEntryResponse",
		map[string]interface{}{
			"status": 0,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode RemoteCarbEntryResponse: %w", err)
	}

	return &Response{
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
func (h *BolusPermissionReleaseHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling BolusPermissionReleaseRequest: txID=%d", msg.TxID)

	log.Info("Releasing bolus permission")

	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"BolusPermissionReleaseResponse",
		map[string]interface{}{
			"status": 0,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode BolusPermissionReleaseResponse: %w", err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// CancelBolusHandler handles CancelBolusRequest messages (used by controlX2)
type CancelBolusHandler struct {
	bridge *pumpx2.Bridge
}

// NewCancelBolusHandler creates a new cancel bolus handler
func NewCancelBolusHandler(bridge *pumpx2.Bridge) *CancelBolusHandler {
	return &CancelBolusHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *CancelBolusHandler) MessageType() string {
	return "CancelBolusRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *CancelBolusHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a CancelBolusRequest
func (h *CancelBolusHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	log.Infof("Handling CancelBolusRequest: txID=%d", msg.TxID)

	if !pumpState.Bolus.Active {
		log.Warn("No active bolus to cancel")
	} else {
		log.Infof("Canceling bolus: delivered %.2f of %.2f units",
			pumpState.Bolus.UnitsDelivered, pumpState.Bolus.UnitsTotal)
	}

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
		"CancelBolusResponse",
		map[string]interface{}{
			"statusId": 0,
			"bolusId":  pumpState.Bolus.BolusID,
			"reasonId": 0,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encode CancelBolusResponse: %w", err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
		StateChanges:    stateChanges,
	}, nil
}
