package harness

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// Clock modes reported and accepted by /api/clock.
const (
	// ClockModeReal is the host wall clock: the emulator's default.
	ClockModeReal = "real"
	// ClockModeManual is a clock the harness sets and steps.
	ClockModeManual = "manual"
)

// clockBody is the request and response shape of /api/clock.
type clockBody struct {
	// Mode is "real" or "manual".
	Mode string `json:"mode"`
	// Now is the clock's current reading, RFC3339. On a PUT it sets the
	// manual clock; it is ignored in real mode.
	Now string `json:"now,omitempty"`
	// PumpOffsetSeconds skews the pump's own clock away from Now. It applies
	// in both modes: a pump whose clock is 8 s fast is a real scenario that
	// has nothing to do with whether the harness is stepping time.
	PumpOffsetSeconds float64 `json:"pump_offset_seconds"`
	// Frozen stops a manual clock where it is.
	Frozen bool `json:"frozen"`
	// PumpNow is the pump's own reading (Now plus the skew), read-only.
	PumpNow string `json:"pump_now,omitempty"`
	// PumpTimeSeconds is PumpNow in the pump-epoch seconds the wire carries,
	// read-only.
	PumpTimeSeconds uint32 `json:"pump_time_seconds,omitempty"`
	// TimeSinceReset is the pump's uptime counter in seconds, read-only.
	TimeSinceReset uint32 `json:"time_since_reset"`
}

func (h *Harness) handleClock(w http.ResponseWriter, r *http.Request) {
	if h.pumpState == nil {
		writeError(w, http.StatusInternalServerError, "no pump state attached")
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, h.clockSnapshot())

	case http.MethodPut, http.MethodPatch, http.MethodPost:
		var body clockBody
		// Start from the current settings so a PUT that names only one field
		// does not silently reset the others.
		body = h.clockSnapshot()
		if err := decodeBody(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, "%v", err)
			return
		}
		if err := h.applyClock(body); err != nil {
			writeError(w, http.StatusBadRequest, "%v", err)
			return
		}
		writeJSON(w, http.StatusOK, h.clockSnapshot())

	default:
		writeError(w, http.StatusMethodNotAllowed, "method %s not allowed on /api/clock", r.Method)
	}
}

// applyClock installs the requested clock mode and skew.
func (h *Harness) applyClock(body clockBody) error {
	h.clockMtx.Lock()
	defer h.clockMtx.Unlock()

	switch body.Mode {
	case ClockModeReal, "":
		if h.manual != nil {
			log.Info("harness: switching the pump back to the real clock")
		}
		h.manual = nil
		h.pumpState.SetClock(state.RealClock{})

	case ClockModeManual:
		now := h.pumpState.Now()
		if body.Now != "" {
			parsed, err := time.Parse(time.RFC3339, body.Now)
			if err != nil {
				return err
			}
			now = parsed
		}
		if h.manual == nil {
			h.manual = state.NewManualClock(now)
			h.pumpState.SetClock(h.manual)
			log.Infof("harness: pump switched to a manual clock at %s", now.Format(time.RFC3339))
		} else {
			h.manual.Set(now)
		}
		h.manual.Freeze(body.Frozen)

	default:
		return fmt.Errorf("unknown clock mode %q (expected %q or %q)", body.Mode, ClockModeReal, ClockModeManual)
	}

	h.pumpState.SetPumpClockOffset(secondsToDuration(body.PumpOffsetSeconds))
	// Refresh the derived uptime counter so a GET right after a PUT is
	// consistent with the new clock.
	h.pumpState.UpdateTimeSinceReset()
	return nil
}

// advanceBody is the request shape of /api/clock/advance.
type advanceBody struct {
	// Seconds moves the clock forward (or, when negative, back).
	Seconds float64 `json:"seconds"`
	// Ticks optionally runs that many simulation steps after the jump instead
	// of the single step advance runs by default. Stepping a long jump in
	// pieces matters when something has to be observed happening partway
	// through it, e.g. a temp rate expiring before a bolus finishes.
	Ticks int `json:"ticks,omitempty"`
}

func (h *Harness) handleClockAdvance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method %s not allowed on /api/clock/advance", r.Method)
		return
	}
	if h.pumpState == nil {
		writeError(w, http.StatusInternalServerError, "no pump state attached")
		return
	}

	var body advanceBody
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}

	h.clockMtx.Lock()
	manual := h.manual
	h.clockMtx.Unlock()

	if manual == nil {
		writeError(w, http.StatusConflict, `the pump is on the real clock; PUT /api/clock {"mode":"manual"} first`)
		return
	}

	total := secondsToDuration(body.Seconds)
	steps := body.Ticks
	if steps < 1 {
		steps = 1
	}

	// Advancing in equal steps and ticking between them lets the simulator see
	// intermediate instants, so events that depend on an ordering within the
	// jump (a temp rate expiring mid-bolus) still happen in order.
	step := total / time.Duration(steps)
	for i := 0; i < steps; i++ {
		if i == steps-1 {
			// Absorb any rounding remainder into the last step.
			manual.Advance(total - step*time.Duration(steps-1))
		} else {
			manual.Advance(step)
		}
		h.pumpState.UpdateTimeSinceReset()
		h.tick()
	}

	log.Infof("harness: advanced the pump clock by %v in %d step(s)", total, steps)
	writeJSON(w, http.StatusOK, h.clockSnapshot())
}

// clockSnapshot reports the clock's current settings.
func (h *Harness) clockSnapshot() clockBody {
	h.clockMtx.Lock()
	manual := h.manual
	h.clockMtx.Unlock()

	mode := ClockModeReal
	frozen := false
	if manual != nil {
		mode = ClockModeManual
		frozen = manual.Frozen()
	}

	return clockBody{
		Mode:              mode,
		Now:               h.pumpState.Now().UTC().Format(time.RFC3339Nano),
		PumpOffsetSeconds: h.pumpState.GetPumpClockOffset().Seconds(),
		Frozen:            frozen,
		PumpNow:           h.pumpState.PumpNow().UTC().Format(time.RFC3339Nano),
		PumpTimeSeconds:   h.pumpState.PumpTimeNow(),
		TimeSinceReset:    h.pumpState.GetTimeSinceReset(),
	}
}
