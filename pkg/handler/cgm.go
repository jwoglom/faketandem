package handler

import (
	"fmt"
	"strings"
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// CGMCommandHandler answers the CGM configuration commands from the pump's
// CGM state (state/cgm.go), which moves the sensor session's lifecycle.
type CGMCommandHandler struct {
	bridge  *pumpx2.Bridge
	msgType string
}

// CGMCommands are the requests CGMCommandHandler answers.
var CGMCommands = []string{
	"SetSensorTypeRequest",
	"SetG6TransmitterIdRequest",
	"StartDexcomG6SensorSessionRequest",
	"SetDexcomG7PairingCodeRequest",
	"StopDexcomCGMSensorSessionRequest",
}

// NewCGMCommandHandler creates the handler for one of CGMCommands.
func NewCGMCommandHandler(bridge *pumpx2.Bridge, msgType string) *CGMCommandHandler {
	return &CGMCommandHandler{bridge: bridge, msgType: msgType}
}

// MessageType returns the message type this handler processes
func (h *CGMCommandHandler) MessageType() string { return h.msgType }

// RequiresAuth returns true
func (h *CGMCommandHandler) RequiresAuth() bool { return true }

// HandleMessage applies the command to the CGM state and answers with its status.
func (h *CGMCommandHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	var status state.CGMCommandStatus
	switch h.msgType {
	case "SetSensorTypeRequest":
		sensorType, _ := cargoInt(msg, "cgmSensorType", "cgmSensorTypeId")
		status = pumpState.CGMSetSensorType(int(sensorType))
	case "SetG6TransmitterIdRequest":
		txID, _ := cargoValue(msg, "txId", "txID")
		status = pumpState.CGMSetG6TransmitterID(strings.TrimSpace(fmt.Sprint(txID)))
	case "StartDexcomG6SensorSessionRequest":
		code, _ := cargoInt(msg, "sensorCode")
		status = pumpState.CGMStartG6Session(int(code))
	case "SetDexcomG7PairingCodeRequest":
		code, ok := cargoInt(msg, "pairingCode")
		if !ok {
			code = -1
		}
		status = pumpState.CGMSetG7PairingCode(int(code))
	case "StopDexcomCGMSensorSessionRequest":
		status = pumpState.CGMStopSession()
	default:
		return nil, fmt.Errorf("CGMCommandHandler: unexpected %s", h.msgType)
	}
	log.Infof("%s: cargo=%v status=%d phase=%s", h.msgType, msg.Cargo, status, pumpState.CGM().Phase)

	params := map[string]interface{}{"status": int(status)}
	if h.msgType == "SetSensorTypeRequest" {
		params["statusAcknowledgement"] = 0
	}
	resType := requestToResponseName(h.msgType)
	response, err := h.bridge.EncodeMessage(msg.TxID, resType, params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", resType, err)
	}
	return &Response{ResponseMessage: response, Immediate: true}, nil
}

// CGMStatusHandler answers the CGM status reads from the pump's CGM state.
type CGMStatusHandler struct {
	bridge  *pumpx2.Bridge
	msgType string
}

// CGMReads are the requests CGMStatusHandler answers.
var CGMReads = []string{
	"CGMStatusRequest",
	"CgmStatusV2Request",
	"CurrentEGVGuiDataRequest",
	"CurrentEgvGuiDataV2Request",
	"GetSavedG7PairingCodeRequest",
	"CGMHardwareInfoRequest",
	"CGMAlertStatusRequest",
}

// NewCGMStatusHandler creates the handler for one of CGMReads.
func NewCGMStatusHandler(bridge *pumpx2.Bridge, msgType string) *CGMStatusHandler {
	return &CGMStatusHandler{bridge: bridge, msgType: msgType}
}

// MessageType returns the message type this handler processes
func (h *CGMStatusHandler) MessageType() string { return h.msgType }

// RequiresAuth returns true
func (h *CGMStatusHandler) RequiresAuth() bool { return true }

// HandleMessage reports the CGM as it is now.
func (h *CGMStatusHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	cgm := pumpState.CGM()
	started := pumpSeconds(pumpState, cgm.SessionStart)

	var params map[string]interface{}
	switch h.msgType {
	case "CGMStatusRequest":
		params = map[string]interface{}{
			"sessionStateId":             cgm.SessionState,
			"lastCalibrationTimestamp":   0,
			"sensorStartedTimestamp":     started,
			"transmitterBatteryStatusId": cgm.TransmitterBattery,
		}
	case "CgmStatusV2Request":
		params = map[string]interface{}{
			"sessionStateId":              cgm.SessionState,
			"lastCalibrationTimestamp":    0,
			"sensorStartedTimestamp":      started,
			"transmitterBatteryStatusId":  cgm.TransmitterBattery,
			"sessionDurationSeconds":      int64(cgm.SessionDuration.Seconds()),
			"sessionTimeRemainingSeconds": int64(cgm.SessionRemaining.Seconds()),
			"cgmSensorTypeId":             cgm.SensorType,
			"gracePeriod":                 cgm.GracePeriod,
		}
	case "CurrentEGVGuiDataRequest", "CurrentEgvGuiDataV2Request":
		params = map[string]interface{}{
			"bgReadingTimestampSeconds": pumpSeconds(pumpState, cgm.EGVTime),
			"cgmReading":                cgm.EGV,
			"egvStatusId":               cgm.EGVStatus,
			"trendRate":                 cgm.TrendRate,
		}
	case "GetSavedG7PairingCodeRequest":
		code := cgm.G7PairingCode
		if code < 0 {
			code = 0
		}
		params = map[string]interface{}{"pairingCode": code}
	case "CGMHardwareInfoRequest":
		info := ""
		if cgm.SensorType == state.CGMSensorTypeDexcomG6 {
			info = cgm.TransmitterID
		}
		params = map[string]interface{}{"hardwareInfoString": info, "lastByte": 0}
	case "CGMAlertStatusRequest":
		params = map[string]interface{}{"cgmAlertBitmask": int64(cgm.AlertBitmask)}
	default:
		return nil, fmt.Errorf("CGMStatusHandler: unexpected %s", h.msgType)
	}

	resType := requestToResponseName(h.msgType)
	response, err := h.bridge.EncodeMessage(msg.TxID, resType, params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", resType, err)
	}
	return &Response{ResponseMessage: response, Immediate: true}, nil
}

// pumpSeconds is t in pump-epoch seconds, or 0 for no time.
func pumpSeconds(pumpState *state.PumpState, t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return int64(pumpState.PumpTimeFor(t))
}
