package harness

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// notificationBody names one notification of the action's category: by its
// pumpX2 enum name in type ("LOW_INSULIN_ALERT", or "LOW_INSULIN" with the
// category's suffix left off), or by its bit in code (bit is an alias).
type notificationBody struct {
	Type    string `json:"type,omitempty"`
	Code    *int   `json:"code,omitempty"`
	Bit     *int   `json:"bit,omitempty"`
	Message string `json:"message,omitempty"`
	// FaultID is a malfunction's fault code, which HighestAam reports beside
	// it ("12-0x2071"). Defaults to defaultMalfunctionFaultID.
	FaultID *uint32 `json:"fault_id,omitempty"`
}

// defaultMalfunctionFaultID is the fault pumpX2's HighestAamResponse documents
// ("12-0x2071"); any value off its ignorable list reads as a malfunction.
const defaultMalfunctionFaultID = 0x2071

func (b notificationBody) bit(c state.NotificationCategory) (uint, error) {
	switch {
	case b.Code != nil:
		return state.ParseNotificationBit(c, strconv.Itoa(*b.Code))
	case b.Bit != nil:
		return state.ParseNotificationBit(c, strconv.Itoa(*b.Bit))
	case b.Type != "":
		return state.ParseNotificationBit(c, b.Type)
	}
	return 0, fmt.Errorf("name the %s with type (its pumpX2 name) or code (its bit)", c)
}

// readNotificationBody reads the body, or for a DELETE with none the type or
// code query parameter.
func readNotificationBody(r *http.Request) (notificationBody, error) {
	var body notificationBody
	if err := decodeBody(r, &body); err != nil {
		return body, err
	}
	query := r.URL.Query()
	if body.Type == "" {
		body.Type = query.Get("type")
	}
	if body.Code == nil && query.Get("code") != "" {
		code, err := strconv.Atoi(query.Get("code"))
		if err != nil {
			return body, fmt.Errorf("code %q is not a number", query.Get("code"))
		}
		body.Code = &code
	}
	return body, nil
}

// notificationRaiseAction is POST /api/state/{alert,alarm,malfunction,reminder,cgmalert}:
// the pump raises the notification, as it would on its own. Its bit goes into
// the category's status bitmask, the activation record is written (none for a
// reminder) and the category's qualifying event goes out.
func (h *Harness) notificationRaiseAction(c state.NotificationCategory) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := readNotificationBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "%v", err)
			return
		}
		bit, err := body.bit(c)
		if err != nil {
			writeError(w, http.StatusBadRequest, "%v", err)
			return
		}
		alert := state.Alert{
			Type:     state.AlertNotification,
			Category: c,
			Bit:      bit,
			Priority: notificationPriority(c),
			Message:  body.Message,
		}
		if c == state.NotificationMalfunction {
			alert.FaultID = defaultMalfunctionFaultID
			if body.FaultID != nil {
				alert.FaultID = *body.FaultID
			}
		}
		if alert.Message == "" {
			alert.Message = state.NotificationName(c, bit)
		}
		raised, ok := h.pumpState.RaiseNotification(alert, "harness")
		if !ok {
			writeError(w, http.StatusConflict, "%s %s already stands", c, state.NotificationName(c, bit))
			return
		}
		if n := h.notifier(); n != nil {
			h.emit(c.String(), func() error { return n.NotifyAlert(raised) })
		}
		log.Infof("harness: raised %s %s (bit %d)", c, raised.Name(), bit)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"raised": alertSnapshotOf(h.pumpState, raised),
			"state":  h.Snapshot(),
		})
	}
}

func notificationPriority(c state.NotificationCategory) state.AlertPriority {
	switch c {
	case state.NotificationAlarm, state.NotificationMalfunction:
		return state.PriorityCritical
	case state.NotificationReminder:
		return state.PriorityInfo
	default:
		return state.PriorityWarning
	}
}

func (h *Harness) notificationClearActions() map[string]http.HandlerFunc {
	return map[string]http.HandlerFunc{
		"alert":       h.notificationClearAction(state.NotificationAlert),
		"alarm":       h.notificationClearAction(state.NotificationAlarm),
		"malfunction": h.notificationClearAction(state.NotificationMalfunction),
		"reminder":    h.notificationClearAction(state.NotificationReminder),
		"cgmalert":    h.notificationClearAction(state.NotificationCGMAlert),
	}
}

// notificationClearAction is DELETE /api/state/{alert,...}: the notification
// is acknowledged on the pump's own screen. It leaves the bitmask, the cleared
// record is written (none for a reminder or a malfunction, which has no such
// record) and the category's qualifying event goes out. A malfunction can be
// cleared only here: a DismissNotificationRequest for one is refused.
func (h *Harness) notificationClearAction(c state.NotificationCategory) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := readNotificationBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "%v", err)
			return
		}
		bit, err := body.bit(c)
		if err != nil {
			writeError(w, http.StatusBadRequest, "%v", err)
			return
		}
		if !h.pumpState.IsNotificationStanding(c, bit) {
			writeError(w, http.StatusConflict, "%s %s is not standing", c, state.NotificationName(c, bit))
			return
		}
		cleared := h.pumpState.ClearNotification(c, bit, h.pumpState.Now())
		for _, a := range cleared {
			h.pumpState.RecordNotificationCleared(a.Alert, a.ClearedAt)
		}
		if n := h.notifier(); n != nil {
			alert := state.Alert{Type: state.AlertNotification, Category: c, Bit: bit}
			h.emit(c.String(), func() error { return n.NotifyAlertCleared(alert) })
		}
		log.Infof("harness: cleared %s %s (bit %d)", c, state.NotificationName(c, bit), bit)
		writeJSON(w, http.StatusOK, map[string]interface{}{"state": h.Snapshot()})
	}
}
