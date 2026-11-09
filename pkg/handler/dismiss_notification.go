package handler

import (
	"fmt"
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// DismissNotificationHandler answers DismissNotificationRequest and takes the
// notification off the pump, as acknowledging it on the pump's screen does:
// its bit leaves the status bitmask of its type (alert, alarm, reminder, CGM
// alert), with a cleared record for each standing one it closes
// (AlertCleared, AlarmCleared, CgmAlertCleared). A malfunction cannot be
// dismissed: a request naming one (type 4, its ActiveAamBits AamType) is
// answered with a nonzero status and the malfunction stays, as does a request
// for a type the pump has none of.
type DismissNotificationHandler struct {
	bridge *pumpx2.Bridge
}

// NewDismissNotificationHandler creates a dismiss notification handler.
func NewDismissNotificationHandler(bridge *pumpx2.Bridge) *DismissNotificationHandler {
	return &DismissNotificationHandler{bridge: bridge}
}

// MessageType returns the message type this handler processes.
func (h *DismissNotificationHandler) MessageType() string {
	return "DismissNotificationRequest"
}

// RequiresAuth returns true if this message requires authentication.
func (h *DismissNotificationHandler) RequiresAuth() bool {
	return true
}

// dismissRefusedStatus is the status a refused dismissal answers with. pumpX2
// names no value; any nonzero one is a failure to TandemKit, and 1 is assumed.
const dismissRefusedStatus = 1

// HandleMessage processes a DismissNotificationRequest.
func (h *DismissNotificationHandler) HandleMessage(msg *pumpx2.ParsedMessage, pumpState *state.PumpState) (*Response, error) {
	id, okID := cargoInt(msg, "notificationId")
	typeID, okType := cargoInt(msg, "notificationTypeId")
	if !okID || !okType {
		return nil, fmt.Errorf("DismissNotificationRequest without a notificationId and notificationTypeId: %v", msg.Cargo)
	}
	log.Infof("Handling DismissNotificationRequest: txID=%d type=%d id=%d", msg.TxID, typeID, id)

	status := 0
	category, known := state.NotificationCategoryFromAamType(int(typeID))
	switch {
	case !known || id < 0 || id > 63:
		log.Warnf("DismissNotificationRequest: no notification type %d id %d to dismiss", typeID, id)
		status = dismissRefusedStatus
	case category == state.NotificationMalfunction:
		log.Warnf("DismissNotificationRequest: malfunction %d cannot be dismissed", id)
		status = dismissRefusedStatus
	case pumpState != nil:
		for _, cleared := range pumpState.ClearNotification(category, uint(id), time.Time{}) {
			pumpState.RecordNotificationCleared(cleared.Alert, cleared.ClearedAt)
		}
	}

	response, err := h.bridge.EncodeMessage(msg.TxID, "DismissNotificationResponse", map[string]interface{}{"status": status})
	if err != nil {
		return nil, fmt.Errorf("failed to encode DismissNotificationResponse: %w", err)
	}
	return &Response{ResponseMessage: response, Immediate: true}, nil
}
