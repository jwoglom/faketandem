package harness

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jwoglom/faketandem/pkg/state"

	log "github.com/sirupsen/logrus"
)

// A scenario is a pump's recent use, replayed before a driver first connects:
// what a phone pairing with a pump that has been delivering for hours finds in
// its history log, its IOB, its last bolus and its basal status.
//
// The replay runs the emulator itself over the past on a frozen clock, starting
// the window's length ago: each bolus, temp rate, suspend and resume goes
// through the same state transitions the protocol and the harness actions use,
// and the clock is stepped through every instant in between, so the history
// records, the 5-minute basal cycle's BasalDelivery and DailyBasal records, the
// reservoir, IOB and the last-bolus record all agree with each other, exactly
// as if the pump had run through them. The clock then goes back to what it was.
// No qualifying events are raised: no driver is connected to receive them.

//go:embed scenarios/*.json
var scenarioPresets embed.FS

// defaultScenarioHours is how much history a scenario replays when it does not
// say: the window TandemKit's setup history load reads.
const defaultScenarioHours = 6

// scenarioStep is the longest the replay moves the clock between two
// simulator ticks.
const scenarioStep = time.Minute

// Scenario event types.
const (
	EventBolus         = "bolus"
	EventTempBasal     = "temp_basal"
	EventTempBasalStop = "temp_basal_stop"
	EventSuspend       = "suspend"
	EventResume        = "resume"
)

// Scenario is the body of POST /api/state/scenario and the file -scenario
// reads.
type Scenario struct {
	// Preset names a built-in scenario (ScenarioPresets); the rest of the body
	// is then ignored.
	Preset string `json:"preset,omitempty"`
	// Name labels the scenario in logs and in the result.
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	// Hours is how far back the replay starts; at least the oldest event's
	// age. Default 6.
	Hours float64 `json:"hours,omitempty"`
	// BasalCycle delivers basal in 5-minute cycles from the window's start and
	// after it, so the log carries a BasalDelivery record every 5 minutes, as a
	// pump's does. Default true.
	BasalCycle *bool `json:"basal_cycle,omitempty"`
	// State is staged at the start of the window, as PUT /api/state stages it:
	// the profile and rates the replay runs on. Reservoir and battery are what
	// the replay then leaves of them.
	State stateUpdate `json:"state"`
	// Events happen in order of age, oldest first.
	Events []ScenarioEvent `json:"events"`
}

// ScenarioEvent is one thing the pump did during the window.
type ScenarioEvent struct {
	// MinutesAgo is how long before the end of the replay (now) it happened.
	MinutesAgo float64 `json:"minutes_ago"`
	// Type is bolus, temp_basal, temp_basal_stop, suspend or resume.
	Type string `json:"type"`

	// Units is a bolus's requested volume, and DeliveredUnits what it put in
	// before it was stopped, when less.
	Units          float64  `json:"units,omitempty"`
	DeliveredUnits *float64 `json:"delivered_units,omitempty"`
	// Source is "pump" (its own screen, the default) or "remote" (an app);
	// for a temp rate, "remote" also writes the SetTempRateResponse record a
	// pump keeps for a temp rate set over Bluetooth. SourceID sets a bolus's
	// pumpX2 BolusSource directly.
	Source   string  `json:"source,omitempty"`
	SourceID *int    `json:"source_id,omitempty"`
	BolusID  *uint32 `json:"bolus_id,omitempty"`
	// DurationSeconds is how long a bolus takes to deliver; default the pump's
	// bolus rate.
	DurationSeconds float64 `json:"duration_seconds,omitempty"`

	// Percent or Rate (U/hr) and DurationMinutes describe a temp rate.
	Percent         *int    `json:"percent,omitempty"`
	Rate            float64 `json:"rate,omitempty"`
	DurationMinutes int     `json:"duration_minutes,omitempty"`
}

// ScenarioResult reports what a replay wrote.
type ScenarioResult struct {
	Name string `json:"name"`
	// Start is the window's first instant on the pump's Clock, End its last.
	Start   string           `json:"start"`
	End     string           `json:"end"`
	Records int              `json:"records"`
	Boluses []ScenarioBolus  `json:"boluses"`
	Temps   []ScenarioTemp   `json:"temp_rates"`
	Final   ScenarioReadings `json:"final"`
}

// ScenarioBolus is one bolus the replay delivered.
type ScenarioBolus struct {
	BolusID        uint32  `json:"bolus_id"`
	RequestedUnits float64 `json:"requested_units"`
	DeliveredUnits float64 `json:"delivered_units"`
	SourceID       int     `json:"source_id"`
	Start          string  `json:"start"`
}

// ScenarioTemp is one temp rate the replay started.
type ScenarioTemp struct {
	TempRateID      int     `json:"temp_rate_id"`
	Percent         int     `json:"percent"`
	Rate            float64 `json:"rate"`
	Start           string  `json:"start"`
	DurationMinutes int     `json:"duration_minutes"`
}

// ScenarioReadings is what a driver reads from the pump once the replay ends.
type ScenarioReadings struct {
	IOB            float64 `json:"iob"`
	ProfileRate    float64 `json:"profile_rate"`
	CurrentRate    float64 `json:"current_rate"`
	TempActive     bool    `json:"temp_active"`
	Suspended      bool    `json:"suspended"`
	LastBolusID    uint32  `json:"last_bolus_id,omitempty"`
	ReservoirUnits float64 `json:"reservoir_units"`
	HistoryCount   int     `json:"history_count"`
}

// ScenarioPresets lists the built-in scenarios by name.
func ScenarioPresets() []string {
	entries, err := scenarioPresets.ReadDir("scenarios")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, strings.TrimSuffix(e.Name(), ".json"))
	}
	sort.Strings(names)
	return names
}

// LoadScenario reads a built-in scenario by name, or a scenario file by path.
func LoadScenario(nameOrPath string) (Scenario, error) {
	data, err := scenarioPresets.ReadFile("scenarios/" + nameOrPath + ".json")
	if err != nil {
		if data, err = os.ReadFile(nameOrPath); err != nil {
			return Scenario{}, fmt.Errorf("%q is neither a preset (%s) nor a readable file: %w",
				nameOrPath, strings.Join(ScenarioPresets(), ", "), err)
		}
	}
	var s Scenario
	if err := json.Unmarshal(data, &s); err != nil {
		return Scenario{}, fmt.Errorf("scenario %s: %w", nameOrPath, err)
	}
	if s.Name == "" {
		s.Name = nameOrPath
	}
	return s, nil
}

// errScenarioConflict is a pump state a scenario cannot be replayed under.
var errScenarioConflict = errors.New("scenario conflict")

func (s Scenario) validate() error {
	for i, e := range s.Events {
		if e.MinutesAgo < 0 {
			return fmt.Errorf("event %d: minutes_ago must not be negative", i)
		}
		switch e.Type {
		case EventBolus:
			if e.Units <= 0 {
				return fmt.Errorf("event %d: a bolus needs positive units", i)
			}
			if e.DeliveredUnits != nil && (*e.DeliveredUnits < 0 || *e.DeliveredUnits > e.Units) {
				return fmt.Errorf("event %d: delivered_units must be between 0 and units", i)
			}
			if _, err := resolveBolusSource(bolusStartBody{Source: e.Source, SourceID: e.SourceID}); err != nil {
				return fmt.Errorf("event %d: %w", i, err)
			}
		case EventTempBasal:
			if (e.Percent == nil) == (e.Rate <= 0) {
				return fmt.Errorf("event %d: a temp_basal needs one of percent and rate", i)
			}
			if e.DurationMinutes <= 0 {
				return fmt.Errorf("event %d: a temp_basal needs a positive duration_minutes", i)
			}
		case EventTempBasalStop, EventSuspend, EventResume:
		default:
			return fmt.Errorf("event %d: unknown type %q (expected %s, %s, %s, %s or %s)", i, e.Type,
				EventBolus, EventTempBasal, EventTempBasalStop, EventSuspend, EventResume)
		}
	}
	return nil
}

// window is how far back the replay starts.
func (s Scenario) window() time.Duration {
	hours := s.Hours
	if hours <= 0 {
		hours = defaultScenarioHours
	}
	window := time.Duration(hours * float64(time.Hour))
	for _, e := range s.Events {
		if age := time.Duration(e.MinutesAgo * float64(time.Minute)); age > window {
			window = age
		}
	}
	return window
}

// replay is one scenario's run over the past.
type replay struct {
	h     *Harness
	ps    *state.PumpState
	clock *state.ManualClock
	// stopAt is when the bolus in progress stops short, zero to run it out.
	stopAt    time.Time
	stopAfter float64
	bolusRate float64
	result    ScenarioResult
}

// ApplyScenario replays s over the pump's past, ending now. It refuses while a
// driver is connected, which could not see the past being rewritten, and while
// a bolus, a temp rate or a suspend is under way now, which the past would
// contradict.
func (h *Harness) ApplyScenario(s Scenario) (ScenarioResult, error) {
	if s.Preset != "" {
		preset, err := LoadScenario(s.Preset)
		if err != nil {
			return ScenarioResult{}, err
		}
		s = preset
	}
	if err := s.validate(); err != nil {
		return ScenarioResult{}, err
	}
	ps := h.pumpState
	switch {
	case h.transport != nil && h.transport.IsConnected():
		return ScenarioResult{}, fmt.Errorf("%w: a driver is connected; disconnect it first", errScenarioConflict)
	case ps.IsBolusActive():
		return ScenarioResult{}, fmt.Errorf("%w: a bolus is in progress", errScenarioConflict)
	case ps.GetTempRate().Active:
		return ScenarioResult{}, fmt.Errorf("%w: a temp rate is running", errScenarioConflict)
	case ps.IsPumpingSuspended():
		return ScenarioResult{}, fmt.Errorf("%w: delivery is suspended", errScenarioConflict)
	}

	events := append([]ScenarioEvent(nil), s.Events...)
	sort.SliceStable(events, func(i, j int) bool { return events[i].MinutesAgo > events[j].MinutesAgo })

	result, err := h.replayScenario(s, events)
	if err != nil {
		return ScenarioResult{}, err
	}
	result.Records = ps.GetHistoryLogCount()
	result.Final = h.scenarioReadings()
	log.Infof("harness: scenario %q wrote %d history records; IOB %.2f U", s.Name, result.Records, result.Final.IOB)
	return result, nil
}

// replayScenario holds the clock for the replay, so no clock request lands in
// the middle of it.
func (h *Harness) replayScenario(s Scenario, events []ScenarioEvent) (ScenarioResult, error) {
	h.clockMtx.Lock()
	defer h.clockMtx.Unlock()

	ps := h.pumpState
	previous := ps.GetClock()
	end := ps.Now()
	start := end.Add(-s.window())
	r := &replay{h: h, ps: ps, clock: state.NewFrozenClock(start), bolusRate: ps.GetBolusRate()}
	r.result = ScenarioResult{Name: s.Name, Start: formatTime(start), End: formatTime(end),
		Boluses: []ScenarioBolus{}, Temps: []ScenarioTemp{}}
	log.Infof("harness: replaying scenario %q from %s", s.Name, start.Format(time.RFC3339))

	ps.SetBasalCycleEnabled(false)
	ps.ForgetPast()
	ps.SetClock(r.clock)
	h.rebaseSimulator()

	staged := s.State
	staged.BasalCycleEnabled = nil
	h.applyStateUpdate(staged)
	if s.BasalCycle == nil || *s.BasalCycle {
		ps.SetBasalCycleEnabled(true)
	}

	var err error
	for _, e := range events {
		r.runTo(end.Add(-time.Duration(e.MinutesAgo * float64(time.Minute))))
		if err = r.apply(e); err != nil {
			break
		}
	}
	if err == nil {
		r.runTo(end)
	}
	ps.SetBolusRate(r.bolusRate)
	// No rebase: the clock goes on from where the replay ended, which is where
	// it was when the replay began.
	ps.SetClock(previous)
	return r.result, err
}

func (h *Harness) rebaseSimulator() {
	if h.simulator != nil {
		h.simulator.Rebase()
	}
}

func (h *Harness) scenarioReadings() ScenarioReadings {
	snap := h.Snapshot()
	readings := ScenarioReadings{
		IOB:            snap.Insulin.IOB,
		ProfileRate:    snap.Basal.ProfileRate,
		CurrentRate:    snap.Basal.CurrentRate,
		TempActive:     snap.Basal.TempActive,
		Suspended:      snap.Basal.Suspended,
		ReservoirUnits: snap.Insulin.ReservoirUnits,
		HistoryCount:   snap.History.Count,
	}
	if snap.LastBolus != nil {
		readings.LastBolusID = snap.LastBolus.BolusID
	}
	return readings
}

// runTo steps the clock to target, ticking the simulator at least every
// scenarioStep and at the instant the bolus in progress ends.
func (r *replay) runTo(target time.Time) {
	for now := r.ps.Now(); now.Before(target); now = r.ps.Now() {
		next := now.Add(scenarioStep)
		if end, ok := r.bolusEnd(); ok && end.Before(next) {
			next = end
		}
		if next.After(target) {
			next = target
		}
		r.clock.Set(next)
		r.ps.UpdateTimeSinceReset()
		r.h.tick()
		if !r.stopAt.IsZero() && !next.Before(r.stopAt) {
			delivered := r.stopAfter
			r.ps.EndBolus(state.BolusEndReasonStopped, &delivered)
			r.stopAt = time.Time{}
		}
	}
}

// bolusEnd is when the bolus in progress finishes or is stopped. A millisecond
// past the arithmetic end, so the simulator's own reading of the elapsed time
// counts the whole bolus delivered.
func (r *replay) bolusEnd() (time.Time, bool) {
	r.ps.RLock()
	bolus := *r.ps.Bolus
	rate := r.ps.BolusRateUnitsPerSecond
	r.ps.RUnlock()
	if !bolus.Active {
		return time.Time{}, false
	}
	if !r.stopAt.IsZero() {
		return r.stopAt, true
	}
	if rate <= 0 {
		rate = state.DefaultBolusRateUnitsPerSecond
	}
	return bolus.StartTime.Add(secondsToDuration(bolus.UnitsTotal/rate) + time.Millisecond), true
}

func (r *replay) apply(e ScenarioEvent) error {
	now := r.ps.Now()
	switch e.Type {
	case EventBolus:
		return r.bolus(e, now)
	case EventTempBasal:
		r.tempBasal(e, now)
	case EventTempBasalStop:
		if _, _, ok := r.ps.EndTempRate(now); !ok {
			return fmt.Errorf("temp_basal_stop %.1f minutes ago: no temp rate is running", e.MinutesAgo)
		}
	case EventSuspend:
		if !r.ps.SuspendDelivery(SuspendReasonUser).Changed {
			return fmt.Errorf("suspend %.1f minutes ago: delivery is already suspended", e.MinutesAgo)
		}
		r.stopAt = time.Time{}
	case EventResume:
		if !r.ps.ResumeDelivery() {
			return fmt.Errorf("resume %.1f minutes ago: delivery is not suspended", e.MinutesAgo)
		}
	}
	return nil
}

func (r *replay) bolus(e ScenarioEvent, now time.Time) error {
	if r.ps.IsBolusActive() {
		return fmt.Errorf("bolus %.1f minutes ago: the bolus before it is still delivering", e.MinutesAgo)
	}
	if r.ps.IsPumpingSuspended() {
		return fmt.Errorf("bolus %.1f minutes ago: delivery is suspended", e.MinutesAgo)
	}
	sourceID, _ := resolveBolusSource(bolusStartBody{Source: e.Source, SourceID: e.SourceID})
	bolusID := r.ps.GetNextBolusID()
	if e.BolusID != nil {
		bolusID = *e.BolusID
	}
	rate := r.bolusRate
	if e.DurationSeconds > 0 {
		rate = e.Units / e.DurationSeconds
	}
	if rate <= 0 {
		rate = state.DefaultBolusRateUnitsPerSecond
	}
	r.ps.SetBolusRate(rate)
	r.ps.StartBolusWithSource(e.Units, bolusID, sourceID, 0)
	r.ps.RecordBolusActivated(bolusID, e.Units, sourceID)
	delivered := e.Units
	if e.DeliveredUnits != nil && *e.DeliveredUnits < e.Units {
		delivered = *e.DeliveredUnits
		r.stopAt = now.Add(secondsToDuration(delivered / rate))
		r.stopAfter = delivered
	}
	r.result.Boluses = append(r.result.Boluses, ScenarioBolus{
		BolusID: bolusID, RequestedUnits: e.Units, DeliveredUnits: delivered, SourceID: sourceID, Start: formatTime(now),
	})
	return nil
}

func (r *replay) tempBasal(e ScenarioEvent, now time.Time) {
	r.ps.EndTempRate(now)
	profileRate := r.ps.GetProfileBasalRate()
	percent, rate := 0, e.Rate
	if e.Percent != nil {
		percent = *e.Percent
		rate = profileRate * float64(percent) / 100
	} else if profileRate > 0 {
		percent = int(math.Round(rate / profileRate * 100))
	}
	basal := &state.BasalState{
		CurrentRate:      profileRate,
		TempBasalActive:  true,
		TempBasalRate:    rate,
		TempBasalEnd:     now.Add(time.Duration(e.DurationMinutes) * time.Minute),
		TempBasalPercent: percent,
		TempBasalStart:   now,
		TempRateID:       r.ps.NextTempRateID(),
	}
	r.ps.SetBasalState(basal)
	r.ps.RecordTempRateActivated(basal, profileRate)
	if strings.EqualFold(e.Source, "remote") {
		r.ps.RecordSetTempRateResponse(0, basal.TempRateID)
	}
	r.result.Temps = append(r.result.Temps, ScenarioTemp{
		TempRateID: basal.TempRateID, Percent: percent, Rate: rate, Start: formatTime(now), DurationMinutes: e.DurationMinutes,
	})
}

// actionScenario replays a scenario over the pump's past: POST
// /api/state/scenario with a Scenario, or {"preset": name}.
func (h *Harness) actionScenario(w http.ResponseWriter, r *http.Request) {
	var body Scenario
	if err := decodeBody(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	result, err := h.ApplyScenario(body)
	switch {
	case errors.Is(err, errScenarioConflict):
		writeError(w, http.StatusConflict, "%v", err)
		return
	case err != nil:
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"scenario": result, "state": h.Snapshot()})
}
