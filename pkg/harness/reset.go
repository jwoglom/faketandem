package harness

import (
	"errors"
	"net/http"

	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// resetPumpSeconds is where a reset leaves the pump's clock: 2008-01-01, one
// second in, as a Mobi woke from shelf mode in a field log. The emulator's clock
// gives no wire reading of 0 in its answer.
const resetPumpSeconds = 1

// errDelivering refuses shelf mode or a reset while the pump delivers: a pump
// goes on the shelf, and comes back from a reset, suspended.
var errDelivering = errors.New("delivery is running; suspend it first")

// enterShelfMode writes the ShelfMode record a pump writes as it goes on the shelf.
func (h *Harness) enterShelfMode() error {
	ps := h.pumpState
	if !ps.IsPumpingSuspended() {
		return errDelivering
	}
	ps.EndTempRate(ps.Now())
	ps.AppendHistory(state.HistoryEvent{TypeID: state.HistoryShelfMode, Name: "ShelfMode", Fields: map[string]interface{}{}})
	return nil
}

// resetPump reboots the pump: its time since reset starts again and it writes
// ArmInit and VersionInfo records. Unless keepClock, its clock restarts at
// 2008-01-01 with no record of the change, and a NewDay record marks the new
// date. Whether a reboot that is not a wake from shelf mode keeps the clock is
// not known; a Mobi woken from shelf mode lost it.
func (h *Harness) resetPump(keepClock bool) error {
	ps := h.pumpState
	if !ps.IsPumpingSuspended() {
		return errDelivering
	}
	ps.EndTempRate(ps.Now())
	if !keepClock {
		prior, after := ps.MovePumpWallClock(resetPumpSeconds)
		log.Infof("harness: pump clock restarted from %d to %d with no record", prior, after)
		ps.AppendHistory(state.HistoryEvent{TypeID: state.HistoryNewDay, Name: "NewDay", Fields: map[string]interface{}{}})
	}
	ps.RestartTimeSinceReset()
	for _, record := range []struct {
		typeID int
		name   string
	}{{state.HistoryArmInit, "ArmInit"}, {state.HistoryVersionInfo, "VersionInfo"}} {
		ps.AppendHistory(state.HistoryEvent{TypeID: record.typeID, Name: record.name, Fields: map[string]interface{}{}})
	}
	return nil
}

type resetBody struct {
	// ShelfMode writes a ShelfMode record first, as a pump woken from the shelf
	// shows in its log.
	ShelfMode bool `json:"shelf_mode,omitempty"`
	// KeepClock leaves the pump's clock where it was.
	KeepClock bool `json:"keep_clock,omitempty"`
}

// actionReset reboots the pump now: POST /api/state/reset. A connected driver
// stays connected, where a pump would drop the link.
func (h *Harness) actionReset(w http.ResponseWriter, r *http.Request) {
	var body resetBody
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	if body.ShelfMode {
		if err := h.enterShelfMode(); err != nil {
			writeError(w, http.StatusConflict, "%v", err)
			return
		}
	}
	if err := h.resetPump(body.KeepClock); err != nil {
		writeError(w, http.StatusConflict, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"clock":   h.clockSnapshot(),
		"history": h.historySnapshot(),
		"state":   h.Snapshot(),
	})
}
