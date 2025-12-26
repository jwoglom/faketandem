package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// CurrentStatusHandler handles CurrentStatusRequest messages
// This provides comprehensive pump status information
type CurrentStatusHandler struct {
	bridge *pumpx2.Bridge
}

// NewCurrentStatusHandler creates a new current status handler
func NewCurrentStatusHandler(bridge *pumpx2.Bridge) *CurrentStatusHandler {
	return &CurrentStatusHandler{
		bridge: bridge,
	}
}

// MessageType returns the message type this handler processes
func (h *CurrentStatusHandler) MessageType() string {
	return "CurrentStatusRequest"
}

// RequiresAuth returns true if this message requires authentication
func (h *CurrentStatusHandler) RequiresAuth() bool {
	return true // Status requires authentication
}

// HandleMessage processes a CurrentStatusRequest
func (h *CurrentStatusHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*HandlerResponse, error) {
	log.Infof("Handling CurrentStatusRequest: txID=%d", msg.TxID)

	// Update time before generating status
	pumpState.UpdateTimeSinceReset()

	// Gather current pump status
	status := h.gatherPumpStatus(pumpState)

	log.Debugf("Current status: reservoir=%.1fu, battery=%d%%, basal=%.2fu/hr, bolus active=%v",
		status["reservoirLevel"], status["batteryPercent"], status["basalRate"], status["bolusActive"])

	// Build response using pumpX2 bridge
	response, err := h.bridge.EncodeMessage(
		msg.TxID,
		"CurrentStatusResponse",
		status,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to encode CurrentStatusResponse: %w", err)
	}

	return &HandlerResponse{
		ResponseMessage: response,
		Characteristic:  bluetooth.CharCurrentStatus, // Use CurrentStatus characteristic
		Immediate:       true,
	}, nil
}

// gatherPumpStatus gathers current pump status from state
func (h *CurrentStatusHandler) gatherPumpStatus(pumpState *state.PumpState) map[string]interface{} {
	pumpState.UpdateTimeSinceReset()

	status := map[string]interface{}{
		// Time
		"timeSinceReset": pumpState.GetTimeSinceReset(),

		// Insulin delivery
		"basalRate":      pumpState.GetBasalRate(),
		"iob":            pumpState.IOB,
		"tdd":            pumpState.TDD,
		"bolusActive":    pumpState.Bolus.Active,
		"bolusDelivered": pumpState.Bolus.UnitsDelivered,
		"bolusTotal":     pumpState.Bolus.UnitsTotal,

		// Physical state
		"reservoirLevel":  pumpState.GetReservoirLevel(),
		"batteryPercent":  pumpState.GetBatteryLevel(),
		"cartridgeAge":    pumpState.Cartridge.DaysSinceChange,

		// Pump info
		"serialNumber": pumpState.GetSerialNumber(),
		"model":        pumpState.Model,

		// Alert count
		"activeAlerts": len(pumpState.ActiveAlerts),
	}

	// Add temp basal info if active
	if pumpState.Basal.TempBasalActive {
		status["tempBasalActive"] = true
		status["tempBasalRate"] = pumpState.Basal.TempBasalRate
	} else {
		status["tempBasalActive"] = false
	}

	return status
}
