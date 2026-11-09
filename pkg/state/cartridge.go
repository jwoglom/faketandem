package state

import (
	"fmt"
	"math"
	"time"
)

// CartridgeSettings are the timings of the cartridge, tubing and cannula procedures. They are
// modeled, not measured on a pump.
type CartridgeSettings struct {
	// ReadySeconds is how long after an accepted EnterChangeCartridgeModeRequest the pump streams
	// readyToChange, the retraction it does first.
	ReadySeconds float64 `json:"ready_seconds"`
	// DetectSeconds is how long the pump takes to check a new cartridge, streaming its progress.
	DetectSeconds float64 `json:"detect_seconds"`
	// DetectOnInsert checks the cartridge as soon as it is put in, under the
	// EnterChangeCartridgeModeRequest's transaction ID, instead of at the ExitChangeCartridgeModeRequest.
	DetectOnInsert bool `json:"detect_on_insert"`
	// FillTubingUnitsPerSecond is how fast the tubing fills while the pump's button is held.
	FillTubingUnitsPerSecond float64 `json:"fill_tubing_units_per_second"`
	// MaxFillSeconds is how long one press fills before the pump stops the fill itself.
	MaxFillSeconds float64 `json:"max_fill_seconds"`
	// FillCannulaStreamLost fills the cannula without streaming FillCannulaStateStreamResponse.
	FillCannulaStreamLost bool `json:"fill_cannula_stream_lost"`
}

// DefaultCartridgeSettings are the settings a new pump starts with.
func DefaultCartridgeSettings() CartridgeSettings {
	return CartridgeSettings{
		ReadySeconds:             5,
		DetectSeconds:            3,
		FillTubingUnitsPerSecond: 0.5,
		MaxFillSeconds:           60,
	}
}

// cartridgeProcedure is where the pump is in a cartridge change, beyond the open workflow mode.
type cartridgeProcedure struct {
	settings CartridgeSettings

	changeTxID    uint8
	readyToChange bool
	// insertedUnits is a new cartridge put in and not yet checked; 0 for none.
	insertedUnits float64
	detecting     bool
	// loaded is a cartridge checked and its tubing not yet filled: LoadStatus's LOAD_CARTRIDGE.
	loaded bool

	fillHeldSince time.Time
	fillPress     uint64
	// fillUnits is what the tubing has taken since fill tubing mode opened.
	fillUnits float64

	cannulaFilling bool
}

// CartridgeSnapshot is the cartridge procedure as the harness reports it.
type CartridgeSnapshot struct {
	LoadState         string            `json:"load_state"`
	LoadingActive     bool              `json:"loading_active"`
	ReadyToChange     bool              `json:"ready_to_change"`
	InsertedUnits     float64           `json:"inserted_units"`
	Detecting         bool              `json:"detecting"`
	Loaded            bool              `json:"loaded"`
	FillTubingUnits   float64           `json:"fill_tubing_units"`
	FillCannulaActive bool              `json:"fill_cannula_active"`
	Settings          CartridgeSettings `json:"settings"`
}

// LoadStatus is the pump's LoadStatusResponse: whether a loading procedure is open, its
// LoadStatusResponse.LoadState ordinal, and the prime status that goes with it.
type LoadStatus struct {
	Active        bool
	LoadStateID   int
	PrimeStatusID int
}

// LoadStatusResponse.LoadState ordinals, and the PrimeTubingStatus ones the emulator reports.
const (
	LoadStateChangeCartridge = 0
	LoadStateLoadCartridge   = 1
	LoadStatePrimeTubing     = 2
	LoadStatePrimeCannula    = 3

	PrimeTubingStatusStart          = 1
	PrimeTubingStatusEnteredCanExit = 2
)

var loadStateNames = map[int]string{
	LoadStateChangeCartridge: "change_cartridge",
	LoadStateLoadCartridge:   "load_cartridge",
	LoadStatePrimeTubing:     "prime_tubing",
	LoadStatePrimeCannula:    "prime_cannula",
}

// CurrentLoadStatus derives LoadStatusResponse from the open procedure.
func (ps *PumpState) CurrentLoadStatus() LoadStatus {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.loadStatusUnlocked()
}

func (ps *PumpState) loadStatusUnlocked() LoadStatus {
	switch {
	case ps.WorkflowModeUnlocked() == WorkflowChangeCartridge:
		return LoadStatus{Active: true, LoadStateID: LoadStateChangeCartridge}
	case ps.WorkflowModeUnlocked() == WorkflowFillTubing:
		prime := PrimeTubingStatusEnteredCanExit
		if ps.fillTubingButtonDown {
			prime = PrimeTubingStatusStart
		}
		return LoadStatus{Active: true, LoadStateID: LoadStatePrimeTubing, PrimeStatusID: prime}
	case ps.cartridge.detecting || ps.cartridge.loaded:
		return LoadStatus{Active: true, LoadStateID: LoadStateLoadCartridge}
	case ps.cartridge.cannulaFilling:
		return LoadStatus{Active: true, LoadStateID: LoadStatePrimeCannula}
	}
	return LoadStatus{}
}

// CartridgeSnapshotUnlocked is for callers already holding the state lock.
func (ps *PumpState) CartridgeSnapshotUnlocked() CartridgeSnapshot {
	c := ps.cartridge
	load := ps.loadStatusUnlocked()
	name := "none"
	if load.Active {
		name = loadStateNames[load.LoadStateID]
	}
	return CartridgeSnapshot{
		LoadState:         name,
		LoadingActive:     load.Active,
		ReadyToChange:     c.readyToChange && ps.WorkflowModeUnlocked() == WorkflowChangeCartridge,
		InsertedUnits:     c.insertedUnits,
		Detecting:         c.detecting,
		Loaded:            c.loaded,
		FillTubingUnits:   c.fillUnits,
		FillCannulaActive: c.cannulaFilling,
		Settings:          ps.cartridgeSettingsUnlocked(),
	}
}

// CartridgeSettings returns the procedures' timings.
func (ps *PumpState) CartridgeSettings() CartridgeSettings {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.cartridgeSettingsUnlocked()
}

func (ps *PumpState) cartridgeSettingsUnlocked() CartridgeSettings {
	if ps.cartridge.settings == (CartridgeSettings{}) {
		return DefaultCartridgeSettings()
	}
	return ps.cartridge.settings
}

// UpdateCartridgeSettings changes the procedures' timings.
func (ps *PumpState) UpdateCartridgeSettings(update func(*CartridgeSettings)) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	s := ps.cartridgeSettingsUnlocked()
	update(&s)
	ps.cartridge.settings = s
}

// SetCartridgeLoaded stages a cartridge checked and waiting for its tubing fill (or clears it),
// with no record and no stream.
func (ps *PumpState) SetCartridgeLoaded(loaded bool) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.cartridge.loaded = loaded
	ps.cartridge.detecting = false
}

// ReservoirCapacity is the most a cartridge holds.
func (ps *PumpState) ReservoirCapacity() float64 {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.Reservoir.MaxUnits
}

// EnterChangeCartridge is EnterWorkflow(WorkflowChangeCartridge) that also keeps txID, which the
// pump streams the change's progress under.
func (ps *PumpState) EnterChangeCartridge(txID uint8) bool {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	if !ps.enterWorkflowUnlocked(WorkflowChangeCartridge) {
		return false
	}
	ps.cartridge.changeTxID = txID
	ps.cartridge.readyToChange = false
	ps.cartridge.insertedUnits = 0
	return true
}

// MarkReadyToChange records that the pump has retracted and the cartridge can come out, and
// returns the EnterChangeCartridgeModeRequest's txID to stream it under.
func (ps *PumpState) MarkReadyToChange() (uint8, error) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	if open := ps.WorkflowModeUnlocked(); open != WorkflowChangeCartridge {
		return 0, fmt.Errorf("the pump is in %s, not %s", open, WorkflowChangeCartridge)
	}
	ps.cartridge.readyToChange = true
	return ps.cartridge.changeTxID, nil
}

// InsertCartridge puts a new cartridge holding units into a pump that is ready for it. With
// DetectOnInsert it starts the check at once and returns detect true and the txID to stream it
// under; otherwise the check waits for ExitChangeCartridgeModeRequest.
func (ps *PumpState) InsertCartridge(units float64) (detect bool, txID uint8, err error) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	if open := ps.WorkflowModeUnlocked(); open != WorkflowChangeCartridge {
		return false, 0, fmt.Errorf("the pump is in %s, not %s", open, WorkflowChangeCartridge)
	}
	if !ps.cartridge.readyToChange {
		return false, 0, fmt.Errorf("the pump has not finished retracting; the cartridge cannot come out yet")
	}
	if ps.cartridge.detecting || ps.cartridge.loaded {
		return false, 0, fmt.Errorf("a cartridge is already in")
	}
	if units <= 0 || units > ps.Reservoir.MaxUnits {
		return false, 0, fmt.Errorf("a cartridge of %g units; the pump takes up to %g", units, ps.Reservoir.MaxUnits)
	}
	ps.cartridge.insertedUnits = units
	if !ps.cartridgeSettingsUnlocked().DetectOnInsert {
		return false, 0, nil
	}
	ps.cartridge.detecting = true
	return true, ps.cartridge.changeTxID, nil
}

// ExitChangeCartridge leaves change cartridge mode, which the pump takes as the new cartridge
// being in: unless one is already checked or being checked, it starts checking it and returns
// detect true.
func (ps *PumpState) ExitChangeCartridge() (accepted, detect bool) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	if !ps.exitWorkflowOpen(WorkflowChangeCartridge) {
		return false, false
	}
	ps.exitWorkflowUnlocked(WorkflowChangeCartridge)
	ps.cartridge.readyToChange = false
	if ps.cartridge.detecting || ps.cartridge.loaded {
		return true, false
	}
	ps.cartridge.detecting = true
	return true, true
}

// FinishCartridgeDetection ends the check of a new cartridge: the reservoir holds what was put in
// (or, with nothing put in, what the cartridge already held) and a CartridgeFilled record is
// written. It reports the units, and false when no check was running.
func (ps *PumpState) FinishCartridgeDetection() (float64, bool) {
	now := ps.Now()
	ps.mutex.Lock()
	if !ps.cartridge.detecting {
		ps.mutex.Unlock()
		return 0, false
	}
	units := ps.cartridge.insertedUnits
	if units <= 0 {
		units = ps.Reservoir.CurrentUnits
	}
	ps.cartridge.detecting = false
	ps.cartridge.loaded = true
	ps.cartridge.insertedUnits = 0
	ps.Reservoir.CurrentUnits = units
	ps.Reservoir.LastFill = now
	ps.Cartridge.DaysSinceChange = 0
	ps.mutex.Unlock()

	ps.AddHistoryLogEntryAt(HistoryCartridgeFilled, "CartridgeFilled", now, map[string]interface{}{
		"insulinDisplay": int(math.Round(units)),
		"insulinActual":  units,
	})
	return units, true
}

// fillTubingButtonUnlocked moves the button and adds what a release ends to the fill. It caps one
// press at MaxFillSeconds, the most the pump fills before stopping on its own.
func (ps *PumpState) fillTubingButtonUnlocked(down bool, now time.Time) {
	c := &ps.cartridge
	switch {
	case down && !ps.fillTubingButtonDown:
		c.fillHeldSince = now
		c.fillPress++
	case !down && ps.fillTubingButtonDown:
		settings := ps.cartridgeSettingsUnlocked()
		held := math.Min(now.Sub(c.fillHeldSince).Seconds(), settings.MaxFillSeconds)
		units := math.Min(math.Max(held, 0)*settings.FillTubingUnitsPerSecond, ps.Reservoir.CurrentUnits)
		c.fillUnits += units
		ps.Reservoir.CurrentUnits -= units
	}
	ps.fillTubingButtonDown = down
}

// FillTubingPress identifies the current press of the button, so a stop timed from one press
// does not release a later one.
func (ps *PumpState) FillTubingPress() uint64 {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	return ps.cartridge.fillPress
}

// ExitFillTubing leaves fill tubing mode, releasing the button, and returns what the tubing took.
// A fill that took any insulin writes a TubingFilled record and completes a cartridge change.
func (ps *PumpState) ExitFillTubing() (accepted bool, units float64) {
	now := ps.Now()
	ps.mutex.Lock()
	if !ps.exitWorkflowOpen(WorkflowFillTubing) {
		ps.mutex.Unlock()
		return false, 0
	}
	ps.fillTubingButtonUnlocked(false, now)
	units = ps.cartridge.fillUnits
	ps.exitWorkflowUnlocked(WorkflowFillTubing)
	if units > 0 {
		ps.cartridge.loaded = false
	}
	ps.mutex.Unlock()

	if units > 0 {
		ps.AddHistoryLogEntryAt(HistoryTubingFilled, "TubingFilled", now, map[string]interface{}{
			"primeSize":        units,
			"completionStatus": FillCompletionStatusCompleted,
			"position":         0,
		})
	}
	return true, units
}

// FillCompletionStatusCompleted is TubingFilled's and CannulaFilled's completionStatus for a fill
// that ran to the end.
const FillCompletionStatusCompleted = 3

// StartFillCannula starts priming the cannula and reports whether the pump accepted it: delivery
// must be suspended, and no other procedure open.
func (ps *PumpState) StartFillCannula() bool {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	if !ps.PumpingSuspended || ps.WorkflowModeUnlocked() != WorkflowNone || ps.cartridge.cannulaFilling {
		return false
	}
	ps.cartridge.cannulaFilling = true
	return true
}

// FinishFillCannula ends a cannula prime of units: the reservoir gives them up and a
// CannulaFilled record is written. It reports false when no prime was running.
func (ps *PumpState) FinishFillCannula(units float64) bool {
	now := ps.Now()
	ps.mutex.Lock()
	if !ps.cartridge.cannulaFilling {
		ps.mutex.Unlock()
		return false
	}
	ps.cartridge.cannulaFilling = false
	ps.Reservoir.CurrentUnits = math.Max(ps.Reservoir.CurrentUnits-units, 0)
	ps.mutex.Unlock()

	ps.AddHistoryLogEntryAt(HistoryCannulaFilled, "CannulaFilled", now, map[string]interface{}{
		"primeSize":        units,
		"completionStatus": FillCompletionStatusCompleted,
		"infusionSetType":  0,
	})
	return true
}
