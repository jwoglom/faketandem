// Package harness is the integration-harness control surface: the HTTP API an
// automated test uses to drive this emulator as a pump under test rather than
// as a demo.
//
// It covers the four things a conformance test needs from the pump side and
// cannot get from the protocol itself:
//
//   - a controllable clock, so pump time can be frozen, stepped or skewed away
//     from wall time and a dose's start and end second can be asserted exactly;
//   - direct reads and writes of pump state, plus pump-initiated actions
//     (start/stall/abort a bolus, start/stop a temp rate, suspend and resume)
//     that move every driver-visible response and raise the qualifying events a
//     real pump would raise;
//   - a request log, the pump's own record of what arrived and what it sent,
//     which is what distinguishes "the pump never answered" from "the driver
//     ignored the answer";
//   - fault injection, so a response can be dropped after the pump has already
//     acted on the request, delayed, replaced with an error, or cut off
//     mid-message.
//
// Every endpoint is inert until used: with nothing armed and no clock set, the
// emulator behaves exactly as it did before this package existed.
package harness

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/faults"
	"github.com/jwoglom/faketandem/pkg/handler"
	"github.com/jwoglom/faketandem/pkg/reqlog"
	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// Harness wires the emulator's moving parts to the control API.
type Harness struct {
	pumpState *state.PumpState
	simulator *state.Simulator
	transport bluetooth.Transport
	router    *handler.Router
	requests  *reqlog.Log
	registry  *faults.Registry

	// fragments applies drop_fragment faults to outgoing notifications, or is
	// nil when the transport cannot filter them.
	fragments *faults.FragmentDropper

	// manual is the manual clock installed while the clock is in manual mode,
	// or nil while the pump runs on the real clock.
	manual   *state.ManualClock
	clockMtx sync.Mutex
}

// Options configures a Harness. Every field except PumpState is optional; a
// missing piece disables the endpoints that need it rather than failing.
type Options struct {
	PumpState *state.PumpState
	Simulator *state.Simulator
	Transport bluetooth.Transport
	Router    *handler.Router
	Requests  *reqlog.Log
	Faults    *faults.Registry
}

// New creates a Harness over the emulator's components.
func New(opts Options) *Harness {
	h := &Harness{
		pumpState: opts.PumpState,
		simulator: opts.Simulator,
		transport: opts.Transport,
		router:    opts.Router,
		requests:  opts.Requests,
		registry:  opts.Faults,
	}
	h.installFragmentDropper()
	return h
}

// installFragmentDropper puts the fragment-level fault seam in place, on a
// transport that has one.
//
// The filter goes in once, at construction, rather than when a drop_fragment
// fault is armed: it consults the registry on every fragment and drops nothing
// while nothing is armed, and installing it lazily would mean a fault armed
// mid-connection started counting fragment positions from the middle of a
// message.
func (h *Harness) installFragmentDropper() {
	if h.registry == nil || h.transport == nil {
		return
	}
	filterer, ok := h.transport.(bluetooth.NotifyFilterer)
	if !ok {
		return
	}

	dropper := faults.NewFragmentDropper(h.registry)
	h.fragments = dropper
	filterer.SetNotifyFilter(func(charType bluetooth.CharacteristicType, data []byte) bool {
		if dropper.Allow(charType.String(), data) {
			return true
		}
		if h.requests != nil {
			h.requests.RecordFault(faults.KindDropFragment, charType.String(), "fragment dropped")
		}
		return false
	})
}

// RegisterRoutes installs the harness endpoints on a mux.
func (h *Harness) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/clock", h.handleClock)
	mux.HandleFunc("/api/clock/advance", h.handleClockAdvance)
	mux.HandleFunc("/api/history", h.handleHistory)
	mux.HandleFunc("/api/state", h.handleState)
	mux.HandleFunc("/api/state/", h.handleStateAction)
	mux.HandleFunc("/api/transport", h.handleTransport)
	mux.HandleFunc("/api/log", h.handleLog)
	mux.HandleFunc("/api/faults", h.handleFaults)
	mux.HandleFunc("/api/faults/", h.handleFaults)
}

// Endpoints returns a human-readable list of the routes this harness serves,
// for the API server's index page.
func (h *Harness) Endpoints() []string {
	return []string{
		"GET    /api/clock",
		"PUT    /api/clock                    {mode, now, pump_offset_seconds, pump_timezone, frozen}",
		"POST   /api/clock/advance            {seconds}",
		"GET    /api/history?since=N&limit=N",
		"GET    /api/state",
		"PUT    /api/state                    (also PATCH) set pump state fields",
		"POST   /api/state/bolus/start        {units, source, duration_seconds|rate, bolus_id}",
		"POST   /api/state/bolus/stall",
		"POST   /api/state/bolus/resume",
		"POST   /api/state/bolus/abort",
		"POST   /api/state/bolus/complete",
		"POST   /api/state/tempbasal/start    {percent, duration_minutes, rate}",
		"POST   /api/state/tempbasal/stop",
		"POST   /api/state/suspend            {reason: user|occlusion|alarm}",
		"POST   /api/state/resume",
		"POST   /api/state/history/append     {type, type_id, seconds_ago, data}",
		"POST   /api/state/qualifyingevent    {bitmask}",
		"GET    /api/transport",
		"PUT    /api/transport                (also PATCH) {att_mtu}",
		"GET    /api/log?since=N",
		"DELETE /api/log",
		"GET    /api/faults",
		"POST   /api/faults                   {kind, message, opcode, count|every, ...}",
		"DELETE /api/faults        (or /api/faults/{id})",
	}
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// writeJSON sends v as the response body.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Errorf("harness: failed to encode response: %v", err)
	}
}

// writeError sends a JSON error body, so a client never has to parse prose.
func writeError(w http.ResponseWriter, status int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Warnf("harness: %s", msg)
	writeJSON(w, status, map[string]string{"error": msg})
}

// decodeBody reads a JSON request body into v. An empty body is accepted and
// leaves v untouched, so actions that take no parameters need no body at all.
func decodeBody(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Debugf("harness: error closing request body: %v", err)
		}
	}()

	if len(strings.TrimSpace(string(body))) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("failed to parse request body: %w", err)
	}
	return nil
}

// notifier returns the qualifying-events notifier, or nil when no router is
// attached (as in a unit test that only exercises state).
func (h *Harness) notifier() *handler.QualifyingEventsNotifier {
	if h.router == nil {
		return nil
	}
	return h.router.GetQualifyingEventsNotifier()
}

// emit sends a qualifying event, logging rather than failing the request when
// no central is subscribed -- a scenario often sets pump state up while
// disconnected, and that is not an error.
func (h *Harness) emit(what string, fn func() error) {
	if fn == nil {
		return
	}
	if err := fn(); err != nil {
		log.Debugf("harness: %s qualifying event not delivered: %v", what, err)
	}
}

// tick runs one simulation step, so an action's consequences (a completed
// bolus, an expired temp rate) are visible in the very next response rather
// than up to a tick later.
func (h *Harness) tick() {
	if h.simulator != nil {
		h.simulator.Tick()
	}
}

// secondsToDuration converts a float seconds value from JSON, preserving
// sub-second precision.
func secondsToDuration(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}
