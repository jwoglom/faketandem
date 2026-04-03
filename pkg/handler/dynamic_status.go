package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// CurrentBolusStatusHandler returns dynamic bolus status from pump state
type CurrentBolusStatusHandler struct {
	bridge *pumpx2.Bridge
}

// NewCurrentBolusStatusHandler creates a new current bolus status handler
func NewCurrentBolusStatusHandler(bridge *pumpx2.Bridge) *CurrentBolusStatusHandler {
	return &CurrentBolusStatusHandler{bridge: bridge}
}

// MessageType returns the message type this handler processes
func (h *CurrentBolusStatusHandler) MessageType() string {
	return "CurrentBolusStatusRequest"
}

// RequiresAuth returns true
func (h *CurrentBolusStatusHandler) RequiresAuth() bool {
	return true
}

// HandleMessage returns dynamic bolus status from pump state
func (h *CurrentBolusStatusHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	pumpState.RLock()
	bolus := pumpState.Bolus
	cargo := map[string]interface{}{}
	if bolus.Active {
		cargo["bolusStatus"] = 1
		cargo["bolusId"] = bolus.BolusID
		cargo["deliveredVolume"] = int(bolus.UnitsDelivered * 1000)
		cargo["totalVolume"] = int(bolus.UnitsTotal * 1000)
		cargo["bolusType"] = 0
	} else {
		cargo["bolusStatus"] = 0
		cargo["bolusId"] = 0
		cargo["deliveredVolume"] = 0
		cargo["totalVolume"] = 0
		cargo["bolusType"] = 0
	}
	pumpState.RUnlock()

	log.Debugf("CurrentBolusStatus: active=%v, bolusId=%v", bolus.Active, cargo["bolusId"])

	response, err := h.bridge.EncodeMessage(msg.TxID, "CurrentBolusStatusResponse", cargo)
	if err != nil {
		return nil, fmt.Errorf("failed to encode CurrentBolusStatusResponse: %w", err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// CurrentBasalStatusHandler returns dynamic basal status from pump state
type CurrentBasalStatusHandler struct {
	bridge *pumpx2.Bridge
}

// NewCurrentBasalStatusHandler creates a new current basal status handler
func NewCurrentBasalStatusHandler(bridge *pumpx2.Bridge) *CurrentBasalStatusHandler {
	return &CurrentBasalStatusHandler{bridge: bridge}
}

// MessageType returns the message type this handler processes
func (h *CurrentBasalStatusHandler) MessageType() string {
	return "CurrentBasalStatusRequest"
}

// RequiresAuth returns true
func (h *CurrentBasalStatusHandler) RequiresAuth() bool {
	return true
}

// HandleMessage returns dynamic basal status from pump state
func (h *CurrentBasalStatusHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	pumpState.RLock()
	basal := pumpState.Basal
	cargo := map[string]interface{}{
		"basalRate":       int(basal.CurrentRate * 1000),
		"tempBasalActive": basal.TempBasalActive,
	}
	if basal.TempBasalActive {
		cargo["tempBasalRate"] = int(basal.TempBasalRate * 1000)
		cargo["tempBasalMinutesRemaining"] = int(basal.TempBasalEnd.Sub(pumpState.CurrentTime).Minutes())
	} else {
		cargo["tempBasalRate"] = 0
		cargo["tempBasalMinutesRemaining"] = 0
	}
	suspended := pumpState.PumpingSuspended
	pumpState.RUnlock()

	cargo["pumpingSuspended"] = suspended

	log.Debugf("CurrentBasalStatus: rate=%v, tempActive=%v, suspended=%v",
		cargo["basalRate"], cargo["tempBasalActive"], suspended)

	response, err := h.bridge.EncodeMessage(msg.TxID, "CurrentBasalStatusResponse", cargo)
	if err != nil {
		return nil, fmt.Errorf("failed to encode CurrentBasalStatusResponse: %w", err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// TempRateStatusHandler returns dynamic temp rate status from pump state
type TempRateStatusHandler struct {
	bridge *pumpx2.Bridge
}

// NewTempRateStatusHandler creates a new temp rate status handler
func NewTempRateStatusHandler(bridge *pumpx2.Bridge) *TempRateStatusHandler {
	return &TempRateStatusHandler{bridge: bridge}
}

// MessageType returns the message type this handler processes
func (h *TempRateStatusHandler) MessageType() string {
	return "TempRateStatusRequest"
}

// RequiresAuth returns true
func (h *TempRateStatusHandler) RequiresAuth() bool {
	return true
}

// HandleMessage returns dynamic temp rate status
func (h *TempRateStatusHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	pumpState.RLock()
	basal := pumpState.Basal
	cargo := map[string]interface{}{
		"tempRateActive": basal.TempBasalActive,
	}
	if basal.TempBasalActive {
		cargo["tempRatePercentage"] = int(basal.TempBasalRate / basal.CurrentRate * 100)
		cargo["tempRateMinutesRemaining"] = int(basal.TempBasalEnd.Sub(pumpState.CurrentTime).Minutes())
	} else {
		cargo["tempRatePercentage"] = 100
		cargo["tempRateMinutesRemaining"] = 0
	}
	pumpState.RUnlock()

	response, err := h.bridge.EncodeMessage(msg.TxID, "TempRateStatusResponse", cargo)
	if err != nil {
		return nil, fmt.Errorf("failed to encode TempRateStatusResponse: %w", err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// InsulinStatusHandler returns dynamic insulin/reservoir status from pump state
type InsulinStatusHandler struct {
	bridge *pumpx2.Bridge
}

// NewInsulinStatusHandler creates a new insulin status handler
func NewInsulinStatusHandler(bridge *pumpx2.Bridge) *InsulinStatusHandler {
	return &InsulinStatusHandler{bridge: bridge}
}

// MessageType returns the message type this handler processes
func (h *InsulinStatusHandler) MessageType() string {
	return "InsulinStatusRequest"
}

// RequiresAuth returns true
func (h *InsulinStatusHandler) RequiresAuth() bool {
	return true
}

// HandleMessage returns dynamic insulin status
func (h *InsulinStatusHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	pumpState.RLock()
	cargo := map[string]interface{}{
		"currentInsulinAmount": int(pumpState.Reservoir.CurrentUnits * 100),
		"maxInsulinAmount":     int(pumpState.Reservoir.MaxUnits * 100),
		"insulinOnBoard":       int(pumpState.IOB * 1000),
	}
	pumpState.RUnlock()

	response, err := h.bridge.EncodeMessage(msg.TxID, "InsulinStatusResponse", cargo)
	if err != nil {
		return nil, fmt.Errorf("failed to encode InsulinStatusResponse: %w", err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// CurrentBatteryHandler returns dynamic battery status from pump state
type CurrentBatteryHandler struct {
	bridge  *pumpx2.Bridge
	msgType string
	resType string
}

// NewCurrentBatteryHandler creates a battery handler for V1 or V2
func NewCurrentBatteryHandler(bridge *pumpx2.Bridge, version string) *CurrentBatteryHandler {
	return &CurrentBatteryHandler{
		bridge:  bridge,
		msgType: "CurrentBattery" + version + "Request",
		resType: "CurrentBattery" + version + "Response",
	}
}

// MessageType returns the message type this handler processes
func (h *CurrentBatteryHandler) MessageType() string {
	return h.msgType
}

// RequiresAuth returns true
func (h *CurrentBatteryHandler) RequiresAuth() bool {
	return true
}

// HandleMessage returns dynamic battery status
func (h *CurrentBatteryHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	pumpState.RLock()
	cargo := map[string]interface{}{
		"batteryLevelPercent": pumpState.Battery.Percentage,
		"isCharging":          pumpState.Battery.Charging,
	}
	pumpState.RUnlock()

	response, err := h.bridge.EncodeMessage(msg.TxID, h.resType, cargo)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", h.resType, err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// ControlIQIOBHandler returns dynamic IOB from pump state
type ControlIQIOBHandler struct {
	bridge  *pumpx2.Bridge
	msgType string
	resType string
}

// NewControlIQIOBHandler creates an IOB handler
func NewControlIQIOBHandler(bridge *pumpx2.Bridge, msgType string) *ControlIQIOBHandler {
	resType := msgType[:len(msgType)-7] + "Response"
	return &ControlIQIOBHandler{bridge: bridge, msgType: msgType, resType: resType}
}

// MessageType returns the message type
func (h *ControlIQIOBHandler) MessageType() string { return h.msgType }

// RequiresAuth returns true
func (h *ControlIQIOBHandler) RequiresAuth() bool { return true }

// HandleMessage returns dynamic IOB
func (h *ControlIQIOBHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	pumpState.RLock()
	cargo := map[string]interface{}{
		"iob":        int(pumpState.IOB * 1000),
		"timeOffset": pumpState.TimeSinceReset,
	}
	pumpState.RUnlock()

	response, err := h.bridge.EncodeMessage(msg.TxID, h.resType, cargo)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", h.resType, err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}
