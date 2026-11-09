package handler

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// NotificationStatusHandler answers the notification status reads from the
// pump's live notification state (state.NotificationBitmask): AlertStatus,
// MalfunctionStatus, ReminderStatus, CGMAlertStatus, HighestAam and
// ActiveAamBits. AlarmStatus has its own handler (alarm_status.go).
type NotificationStatusHandler struct {
	bridge  *pumpx2.Bridge
	msgType string
}

// NotificationStatusRequests are the requests NotificationStatusHandler answers.
var NotificationStatusRequests = []string{
	"AlertStatusRequest",
	"MalfunctionStatusRequest",
	"ReminderStatusRequest",
	"CGMAlertStatusRequest",
	"HighestAamRequest",
	"ActiveAamBitsRequest",
}

// NewNotificationStatusHandler creates the handler for one of NotificationStatusRequests.
func NewNotificationStatusHandler(bridge *pumpx2.Bridge, msgType string) *NotificationStatusHandler {
	return &NotificationStatusHandler{bridge: bridge, msgType: msgType}
}

// MessageType returns the message type this handler processes.
func (h *NotificationStatusHandler) MessageType() string { return h.msgType }

// RequiresAuth returns true.
func (h *NotificationStatusHandler) RequiresAuth() bool { return true }

// HandleMessage answers from the live bitmasks.
func (h *NotificationStatusHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	if h.msgType == "ReminderStatusRequest" {
		mask := pumpState.NotificationBitmask(state.NotificationReminder)
		log.Infof("Handling ReminderStatusRequest: txID=%d, reminderBitmask=0x%016x", msg.TxID, mask)
		response, err := protocol.BuildReminderStatusResponse(uint8(msg.TxID), mask)
		if err != nil {
			return nil, fmt.Errorf("failed to build ReminderStatusResponse: %w", err)
		}
		return &Response{NativeResponse: response, Immediate: true}, nil
	}

	name, params := h.responseParams(msg, pumpState)
	log.Infof("Handling %s: txID=%d, %s %v", h.msgType, msg.TxID, name, params)
	response, err := h.bridge.EncodeMessage(msg.TxID, name, params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode %s: %w", name, err)
	}
	return &Response{ResponseMessage: response, Immediate: true}, nil
}

func (h *NotificationStatusHandler) responseParams(msg *pumpx2.ParsedMessage, ps *state.PumpState) (string, map[string]interface{}) {
	switch h.msgType {
	case "AlertStatusRequest":
		// Raw cargo: the BigInteger constructor is out of a JSON number's reach.
		return "AlertStatusResponse", map[string]interface{}{"raw": bitmaskHex(ps.NotificationBitmask(state.NotificationAlert))}
	case "MalfunctionStatusRequest":
		mask := ps.NotificationBitmask(state.NotificationMalfunction)
		return "MalfunctionBitmaskStatusResponse", map[string]interface{}{
			"codeA": int64(uint32(mask)),
			"codeB": int64(uint32(mask >> 32)),
		}
	case "CGMAlertStatusRequest":
		return "CGMAlertStatusResponse", map[string]interface{}{
			"cgmAlertBitmask": int64(ps.NotificationBitmask(state.NotificationCGMAlert)),
		}
	case "HighestAamRequest":
		aamID, faultID := int64(0), int64(0)
		if highest, ok := ps.HighestNotification(); ok {
			aamID, faultID = int64(highest.Bit), int64(highest.FaultID)
		}
		return "HighestAamResponse", map[string]interface{}{"aamId": aamID, "faultId": faultID, "remaining": "000000"}
	default: // ActiveAamBitsRequest
		return "ActiveAamBitsResponse", map[string]interface{}{"raw": activeAamBitsCargo(msg, ps)}
	}
}

// activeAamBitsCargo is the ActiveAamBitsResponse cargo for the AamType the
// request's one byte asks for (TandemKit sends 2 for alarms and 4 for
// malfunctions), or, with none, for the highest category standing. Every
// standing notification is unacknowledged: acknowledging one takes it off.
func activeAamBitsCargo(msg *pumpx2.ParsedMessage, ps *state.PumpState) string {
	category, ok := state.NotificationNone, false
	if raw, has := cargoValue(msg, "cargo"); has {
		if s, isString := raw.(string); isString {
			if b, err := hex.DecodeString(s); err == nil && len(b) == 1 {
				category, ok = state.NotificationCategoryFromAamType(int(b[0]))
			}
		}
	}
	if !ok {
		if highest, standing := ps.HighestNotification(); standing {
			category, ok = highest.Category, true
		}
	}
	cargo := make([]byte, 17)
	cargo[16] = 5 // NONE
	if ok {
		mask := ps.NotificationBitmask(category)
		binary.LittleEndian.PutUint64(cargo[0:8], mask)
		binary.LittleEndian.PutUint64(cargo[8:16], mask)
		cargo[16] = byte(category.AamType())
	}
	return hex.EncodeToString(cargo)
}

func bitmaskHex(mask uint64) string {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, mask)
	return hex.EncodeToString(b)
}
