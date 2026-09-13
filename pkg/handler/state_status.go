package handler

import (
	"fmt"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// LastBolusStatusHandler answers LastBolusStatusRequest / LastBolusStatusV2Request
// / LastBolusStatusV3Request from live pump state.
//
// These were previously served as static all-zero constants from the settings
// table. A driver reads this message to reconcile a bolus it commanded with
// what the pump actually delivered: it matches the reported bolusId against the
// one it was given by BolusPermissionResponse, and takes deliveredVolume and
// the end timestamp from here. Against a constant bolusId of 0 every completion
// looks like "some other bolus", so the driver holds its dose unfinalized and
// keeps retrying. Serving the real record is what makes a bolus initiated via
// InitiateBolusRequest observable afterwards.
type LastBolusStatusHandler struct {
	bridge  *pumpx2.Bridge
	msgType string
	resType string
}

// NewLastBolusStatusHandler creates a handler for one of the LastBolusStatus
// request variants ("LastBolusStatusRequest", "LastBolusStatusV2Request" or
// "LastBolusStatusV3Request").
func NewLastBolusStatusHandler(bridge *pumpx2.Bridge, msgType string) *LastBolusStatusHandler {
	return &LastBolusStatusHandler{
		bridge:  bridge,
		msgType: msgType,
		resType: requestToResponseName(msgType),
	}
}

// MessageType returns the message type this handler processes
func (h *LastBolusStatusHandler) MessageType() string { return h.msgType }

// RequiresAuth returns true
func (h *LastBolusStatusHandler) RequiresAuth() bool { return true }

// HandleMessage builds a LastBolusStatus response from the recorded last bolus.
func (h *LastBolusStatusHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	record, ok := pumpState.GetLastBolus()
	if !ok {
		log.Debugf("%s: no bolus has finished yet, reporting an empty record", h.msgType)
	}

	// All volumes on the wire are milliunits; timestamps are pump-epoch seconds.
	deliveredMilliunits := int64(record.DeliveredUnits * 1000)
	requestedMilliunits := int64(record.RequestedUnits * 1000)
	timestamp := uint32(0)
	if ok {
		timestamp = pumpState.PumpTimeFor(record.EndTime)
	}

	var cargo map[string]interface{}
	switch h.resType {
	case "LastBolusStatusV3Response":
		// V3 splits the record into a standard and an extended section, with a
		// presence bitmask in byte 0 (bit 0 = standard present, bit 1 =
		// extended). faketandem does not simulate extended boluses, so only the
		// standard section is ever reported present.
		presence := 0
		if ok {
			presence = 0x01
		}
		cargo = map[string]interface{}{
			"lastBolusTypeBitmask":               presence,
			"standardBolusStatusId":              0,
			"standardBolusId":                    record.BolusID,
			"standardUnknown":                    "0000",
			"standardBolusTimestamp":             timestamp,
			"standardBolusDeliveredVolume":       deliveredMilliunits,
			"standardBolusEndReasonId":           record.EndReasonID,
			"standardBolusSourceId":              record.SourceID,
			"standardBolusTypeBitmask":           record.TypeBitmask,
			"standardBolusRequestedVolume":       requestedMilliunits,
			"standardBolusSecondsSincePumpReset": record.SecondsSinceReset,
			"extendedBolusStatusId":              0,
			"extendedBolusId":                    0,
			"extendedUnknown":                    "0000",
			"extendedBolusTimestamp":             0,
			"extendedBolusDeliveredVolume":       0,
			"extendedBolusEndReasonId":           0,
			"extendedBolusSourceId":              0,
			"extendedBolusTypeBitmask":           0,
			"extendedBolusRequestedVolume":       0,
			"extendedBolusSecondsSincePumpReset": 0,
			"extendedBolusDuration":              0,
		}
	case "LastBolusStatusV2Response":
		cargo = map[string]interface{}{
			"status":                0,
			"bolusId":               record.BolusID,
			"timestamp":             timestamp,
			"deliveredVolume":       deliveredMilliunits,
			"bolusStatusId":         record.EndReasonID,
			"bolusSourceId":         record.SourceID,
			"bolusTypeBitmask":      record.TypeBitmask,
			"extendedBolusDuration": 0,
			"requestedVolume":       requestedMilliunits,
		}
	default:
		// LastBolusStatusResponse (V1): same shape as V2 except the trailing
		// requestedVolume is an opaque 2-byte "unknown" field instead.
		cargo = map[string]interface{}{
			"status":                0,
			"bolusId":               record.BolusID,
			"timestamp":             timestamp,
			"deliveredVolume":       deliveredMilliunits,
			"bolusStatusId":         record.EndReasonID,
			"bolusSourceId":         record.SourceID,
			"bolusTypeBitmask":      record.TypeBitmask,
			"extendedBolusDuration": 0,
			"unknown":               "0000",
		}
	}

	log.Debugf("%s: bolusId=%d delivered=%d mU of %d mU endReason=%d timestamp=%d",
		h.resType, record.BolusID, deliveredMilliunits, requestedMilliunits, record.EndReasonID, timestamp)

	response, err := h.bridge.EncodeMessage(msg.TxID, h.resType, cargo)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", h.resType, err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// TempRateHandler answers TempRateRequest from live pump state.
//
// Drivers read this before suspending (and before enacting a new temp rate) to
// decide whether an existing temp rate has to be stopped first. Serving the
// static {active:false} constant it used to meant a temp rate set through
// SetTempRateRequest was invisible, so the driver never stopped it.
type TempRateHandler struct {
	bridge *pumpx2.Bridge
}

// NewTempRateHandler creates a new temp rate status handler
func NewTempRateHandler(bridge *pumpx2.Bridge) *TempRateHandler {
	return &TempRateHandler{bridge: bridge}
}

// MessageType returns the message type this handler processes
func (h *TempRateHandler) MessageType() string { return "TempRateRequest" }

// RequiresAuth returns true
func (h *TempRateHandler) RequiresAuth() bool { return true }

// HandleMessage builds a TempRateResponse from the current temp basal state.
func (h *TempRateHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	temp := pumpState.GetTempRate()

	// TempRateResponse(boolean active, int percentage, long startTimeRaw,
	// long duration). startTimeRaw is pump-epoch seconds.
	//
	// TODO: duration's unit is not documented by either pumpX2 or TandemKit,
	// and no capture of an active temp rate exists to settle it. Seconds is
	// used here for consistency with startTimeRaw; note that the *request*
	// side (SetTempRateRequest) carries its duration in milliseconds, so a real
	// capture during an active temp rate should confirm this before anything
	// depends on it. No current driver code reads the field -- only `active`.
	startTimeRaw := uint32(0)
	durationSeconds := int64(0)
	if temp.Active {
		startTimeRaw = pumpState.PumpTimeFor(temp.StartTime)
		if !temp.EndTime.IsZero() && !temp.StartTime.IsZero() {
			durationSeconds = int64(temp.EndTime.Sub(temp.StartTime).Seconds())
		}
	}

	cargo := map[string]interface{}{
		"active":       temp.Active,
		"percentage":   temp.Percent,
		"startTimeRaw": startTimeRaw,
		"duration":     durationSeconds,
	}

	log.Debugf("TempRateResponse: active=%v percentage=%d startTimeRaw=%d duration=%ds",
		temp.Active, temp.Percent, startTimeRaw, durationSeconds)

	response, err := h.bridge.EncodeMessage(msg.TxID, "TempRateResponse", cargo)
	if err != nil {
		return nil, fmt.Errorf("failed to encode TempRateResponse: %w", err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// ControlIQInfoHandler answers ControlIQInfoV1Request / ControlIQInfoV2Request
// from live pump state.
//
// The static default used to report closedLoopEnabled=true, which makes a
// driver set "use Control-IQ" on its first sync and then refuse to enact temp
// basals or manual boluses at all -- so most of the control surface was
// untestable without a settings override before pairing. It now reflects
// PumpState.ClosedLoopEnabled, which defaults to false.
type ControlIQInfoHandler struct {
	bridge  *pumpx2.Bridge
	msgType string
	resType string
}

// NewControlIQInfoHandler creates a handler for ControlIQInfoV1Request or
// ControlIQInfoV2Request.
func NewControlIQInfoHandler(bridge *pumpx2.Bridge, msgType string) *ControlIQInfoHandler {
	return &ControlIQInfoHandler{
		bridge:  bridge,
		msgType: msgType,
		resType: requestToResponseName(msgType),
	}
}

// MessageType returns the message type this handler processes
func (h *ControlIQInfoHandler) MessageType() string { return h.msgType }

// RequiresAuth returns true
func (h *ControlIQInfoHandler) RequiresAuth() bool { return true }

// HandleMessage builds a ControlIQInfo response from pump state.
func (h *ControlIQInfoHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	info := pumpState.GetControlIQInfo()

	// ControlIQInfoV1Response(boolean closedLoopEnabled, int weight,
	// int weightUnit, int totalDailyInsulin, int currentUserModeType,
	// int byte6, int byte7, int byte8, int controlStateType)
	cargo := map[string]interface{}{
		"closedLoopEnabled":   info.ClosedLoopEnabled,
		"weight":              info.Weight,
		"weightUnit":          0,
		"totalDailyInsulin":   info.TotalDailyInsulin,
		"currentUserModeType": info.CurrentUserModeType,
		"byte6":               0,
		"byte7":               0,
		"byte8":               0,
		"controlStateType":    0,
	}
	if h.resType == "ControlIQInfoV2Response" {
		// V2 adds the exercise-mode fields.
		cargo["exerciseChoice"] = 0
		cargo["exerciseDuration"] = 0
		cargo["exerciseTimeRemaining"] = 0
	}

	log.Debugf("%s: closedLoopEnabled=%v userModeType=%d", h.resType, info.ClosedLoopEnabled, info.CurrentUserModeType)

	response, err := h.bridge.EncodeMessage(msg.TxID, h.resType, cargo)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", h.resType, err)
	}

	return &Response{
		ResponseMessage: response,
		Immediate:       true,
	}, nil
}

// requestToResponseName turns a "…Request" message name into its "…Response"
// counterpart, leaving anything else untouched.
func requestToResponseName(msgType string) string {
	const suffix = "Request"
	if len(msgType) > len(suffix) && msgType[len(msgType)-len(suffix):] == suffix {
		return msgType[:len(msgType)-len(suffix)] + "Response"
	}
	return msgType
}
