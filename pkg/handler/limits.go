package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// DeliveryLimitsReadHandler answers GlobalMaxBolusSettingsRequest and BasalLimitSettingsRequest
// from the pump's delivery limits.
type DeliveryLimitsReadHandler struct {
	bridge  *pumpx2.Bridge
	msgType string
}

// NewDeliveryLimitsReadHandler creates a handler for one of the two limit reads.
func NewDeliveryLimitsReadHandler(bridge *pumpx2.Bridge, msgType string) *DeliveryLimitsReadHandler {
	return &DeliveryLimitsReadHandler{bridge: bridge, msgType: msgType}
}

// MessageType returns the message type this handler processes
func (h *DeliveryLimitsReadHandler) MessageType() string { return h.msgType }

// RequiresAuth returns true
func (h *DeliveryLimitsReadHandler) RequiresAuth() bool { return true }

// HandleMessage answers with the limit the pump holds now.
func (h *DeliveryLimitsReadHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	limits := pumpState.GetDeliveryLimits()
	var responseType string
	var params map[string]interface{}
	switch h.msgType {
	case "GlobalMaxBolusSettingsRequest":
		responseType = "GlobalMaxBolusSettingsResponse"
		params = map[string]interface{}{
			"maxBolus":        limits.MaxBolusMilliunits,
			"maxBolusDefault": limits.MaxBolusDefaultMilliunits,
		}
	case "BasalLimitSettingsRequest":
		responseType = "BasalLimitSettingsResponse"
		params = map[string]interface{}{
			"basalLimit":        limits.MaxBasalMilliunits,
			"basalLimitDefault": limits.MaxBasalDefaultMilliunits,
		}
	default:
		return nil, fmt.Errorf("no delivery limit read for %s", h.msgType)
	}
	response, err := h.bridge.EncodeMessage(msg.TxID, responseType, params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", responseType, err)
	}
	return &Response{ResponseMessage: response, Immediate: true}, nil
}

// DeliveryLimitWriteHandler applies SetMaxBolusLimitRequest and SetMaxBasalLimitRequest to the
// pump's delivery limits.
type DeliveryLimitWriteHandler struct {
	bridge  *pumpx2.Bridge
	msgType string
}

// NewDeliveryLimitWriteHandler creates a handler for one of the two limit writes.
func NewDeliveryLimitWriteHandler(bridge *pumpx2.Bridge, msgType string) *DeliveryLimitWriteHandler {
	return &DeliveryLimitWriteHandler{bridge: bridge, msgType: msgType}
}

// MessageType returns the message type this handler processes
func (h *DeliveryLimitWriteHandler) MessageType() string { return h.msgType }

// RequiresAuth returns true
func (h *DeliveryLimitWriteHandler) RequiresAuth() bool { return true }

// HandleMessage stores the new limit. pumpX2's fields are maxBolusMilliunits and
// maxHourlyBasalMilliunits; the reads answer them as maxBolus and basalLimit.
func (h *DeliveryLimitWriteHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	var responseType string
	switch h.msgType {
	case "SetMaxBolusLimitRequest":
		milliunits, ok := cargoInt(msg, "maxBolusMilliunits")
		if !ok {
			return nil, fmt.Errorf("SetMaxBolusLimitRequest without maxBolusMilliunits: %v", msg.Cargo)
		}
		pumpState.SetMaxBolusMilliunits(int(milliunits))
		responseType = "SetMaxBolusLimitResponse"
		log.Infof("Max bolus set to %d mU", milliunits)
	case "SetMaxBasalLimitRequest":
		milliunits, ok := cargoInt(msg, "maxHourlyBasalMilliunits")
		if !ok {
			return nil, fmt.Errorf("SetMaxBasalLimitRequest without maxHourlyBasalMilliunits: %v", msg.Cargo)
		}
		pumpState.SetMaxBasalMilliunits(int(milliunits))
		responseType = "SetMaxBasalLimitResponse"
		log.Infof("Max basal set to %d mU/h", milliunits)
	default:
		return nil, fmt.Errorf("no delivery limit write for %s", h.msgType)
	}
	response, err := h.bridge.EncodeMessage(msg.TxID, responseType, map[string]interface{}{"status": 0})
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", responseType, err)
	}
	return &Response{ResponseMessage: response, Immediate: true}, nil
}
