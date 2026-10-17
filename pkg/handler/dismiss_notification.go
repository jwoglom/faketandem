package handler

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/settings"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// DismissNotificationHandler answers DismissNotificationRequest and takes the
// notification off the pump, as acknowledging it on the pump's screen does: an
// alarm leaves AlarmStatus, with an AlarmCleared record for each standing alarm
// it closes, and an alert leaves AlertStatus. Reminders and CGM alerts are
// identified by an ID the emulator does not model, so they are only logged.
type DismissNotificationHandler struct {
	bridge          *pumpx2.Bridge
	settingsManager *settings.Manager
}

// NewDismissNotificationHandler creates a dismiss notification handler.
func NewDismissNotificationHandler(bridge *pumpx2.Bridge, settingsManager *settings.Manager) *DismissNotificationHandler {
	return &DismissNotificationHandler{bridge: bridge, settingsManager: settingsManager}
}

// MessageType returns the message type this handler processes.
func (h *DismissNotificationHandler) MessageType() string {
	return "DismissNotificationRequest"
}

// RequiresAuth returns true if this message requires authentication.
func (h *DismissNotificationHandler) RequiresAuth() bool {
	return true
}

// HandleMessage processes a DismissNotificationRequest.
func (h *DismissNotificationHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	id, okID := cargoInt(msg, "notificationId")
	typeID, okType := cargoInt(msg, "notificationTypeId")
	if !okID || !okType {
		return nil, fmt.Errorf("DismissNotificationRequest without a notificationId and notificationTypeId: %v", msg.Cargo)
	}
	log.Infof("Handling DismissNotificationRequest: txID=%d type=%d id=%d", msg.TxID, typeID, id)

	switch control.NotificationType(typeID) {
	case control.NotificationTypeAlarm:
		if pumpState != nil && id >= 0 && id <= 63 {
			for _, cleared := range pumpState.DismissAlarm(uint(id), time.Time{}) {
				pumpState.RecordAlarmCleared(cleared.Alert, cleared.ClearedAt)
			}
		}
	case control.NotificationTypeAlert:
		if err := h.dismissAlert(id); err != nil {
			log.Warnf("DismissNotificationRequest: alert %d left standing: %v", id, err)
		}
	default:
		log.Infof("DismissNotificationRequest: type %d is not modeled; nothing to clear", typeID)
	}

	response, err := h.bridge.EncodeMessage(msg.TxID, "DismissNotificationResponse", map[string]interface{}{"status": 0})
	if err != nil {
		return nil, fmt.Errorf("failed to encode DismissNotificationResponse: %w", err)
	}
	return &Response{ResponseMessage: response, Immediate: true}, nil
}

// dismissAlert clears bit id of AlertStatusRequest's constant raw cargo, an
// eight-byte little-endian bitmask (see settings.RegisterDefaults).
func (h *DismissNotificationHandler) dismissAlert(id int64) error {
	if h.settingsManager == nil || id < 0 || id > 63 {
		return fmt.Errorf("no alert bit %d", id)
	}
	config, err := h.settingsManager.GetConfig("AlertStatusRequest")
	if err != nil {
		return err
	}
	if config.Mode != settings.ModeConstant {
		return fmt.Errorf("AlertStatusRequest is in %s mode", config.Mode)
	}
	rawHex, _ := config.Value["raw"].(string)
	raw, err := hex.DecodeString(rawHex)
	if err != nil || len(raw) != 8 {
		return fmt.Errorf("AlertStatusRequest raw %q is not an eight-byte bitmask", rawHex)
	}
	raw[id/8] &^= 1 << uint(id%8)
	return h.settingsManager.UpdateConstant("AlertStatusRequest", map[string]interface{}{"raw": hex.EncodeToString(raw)})
}
