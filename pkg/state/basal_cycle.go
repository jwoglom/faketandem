package state

import (
	"math"
	"sort"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// The basal cycle.
//
// A Tandem pump does not deliver basal continuously. Every five minutes it
// writes a BasalDelivery (279) record and delivers that cycle's 300 seconds at
// the record's commandedRate. A change of rate -- a temp rate starting or
// ending, a profile segment boundary, Control-IQ -- takes effect at the next
// 279, so a temp rate set and replaced between two 279s is never delivered.
// CurrentBasalStatus shows a command at once; only delivery waits.
//
// This is what TandemKit's basal reconciliation reads back (TandemKit#514), so
// the emulator models it when basal_cycle_enabled is set, and otherwise keeps
// its old continuous delivery: drivers' existing tests stage their own 279s.
//
// Time-driven events (each 279, a temp rate expiring, the daily total growing,
// a DailyBasal record) are processed in time order up to the instant of every
// change that could move them -- a command, a tick, a profile write -- so a
// record lands in the log at its own instant, in sequence with everything
// else, however coarse the ticks are.

// BasalCycleLength is the time between two 279 records.
const BasalCycleLength = 5 * time.Minute

const (
	// dailyBasalGrowthDelay is how long after its 279 a cycle's insulin is in
	// DailyBasal.dailyTotalBasal on a Mobi.
	dailyBasalGrowthDelay = 10 * time.Second
	// dailyBasalRecordDelay is when the emulator writes the DailyBasal record
	// for a cycle: late enough that a reader taking the total as settled a
	// minute after the 279 (as TandemKit does) can use it.
	dailyBasalRecordDelay = 90 * time.Second
)

// commandedRateSource values a 279 carries.
const (
	BasalSourceSuspended = 0
	BasalSourceProfile   = 1
	BasalSourceTemp      = 2
	BasalSourceControlIQ = 3
)

// currentPcm values a ControlIQPcmChange (230) carries.
const (
	PCMNoControl  = 0 // suspended
	PCMOpenLoop   = 1 // Control-IQ off
	PCMNoCGM      = 2 // Control-IQ on without a CGM: the profile runs
	PCMClosedLoop = 3
)

type basalCycleEventKind int

const (
	cycleGrowth basalCycleEventKind = iota
	cycleDailyRecord
)

type basalCycleEvent struct {
	at    time.Time
	kind  basalCycleEventKind
	units float64
}

// basalCycle is the cycle's bookkeeping. mtx is taken before the pump state's
// own mutex, never after.
type basalCycle struct {
	mtx sync.Mutex

	enabled bool
	// through is the instant the cycle has been simulated to.
	through time.Time
	// next is when the next 279 is due.
	next time.Time
	// latest is the latest 279's instant, and rate and source what it said.
	latest time.Time
	rate   float64
	source int

	// crossedWhileSuspended decides whether a resume writes a 279: one after
	// a suspend that spans a cycle boundary does, one within the suspend's own
	// cycle does not.
	crossedWhileSuspended bool

	events     []basalCycleEvent
	dailyTotal float64
	dailyDay   int64

	pcm           int
	cgmAvailable  bool
	algorithmRate float64 // U/hr; negative means the profile rate

	// segment is the active profile's segment last put in force, so a rate
	// staged with basal_rate holds until the next segment boundary.
	segment int
}

func newBasalCycle() *basalCycle {
	return &basalCycle{pcm: PCMOpenLoop, cgmAvailable: true, algorithmRate: -1, dailyDay: -1, segment: -1}
}

// BasalCycleSnapshot is the cycle's state as the harness reports it.
type BasalCycleSnapshot struct {
	Enabled bool
	// Next is when the next 279 is due, and Latest the latest 279's instant.
	Next            time.Time
	Latest          time.Time
	Rate            float64
	Source          int
	DailyTotalBasal float64
	PCM             int
	CGMAvailable    bool
	// AlgorithmRate is Control-IQ's scripted rate, negative for the profile rate.
	AlgorithmRate float64
}

// GetBasalCycle reports the cycle's state.
func (ps *PumpState) GetBasalCycle() BasalCycleSnapshot {
	c := ps.basalCycle
	c.mtx.Lock()
	defer c.mtx.Unlock()
	return BasalCycleSnapshot{
		Enabled:         c.enabled,
		Next:            c.next,
		Latest:          c.latest,
		Rate:            c.rate,
		Source:          c.source,
		DailyTotalBasal: c.dailyTotal,
		PCM:             c.pcm,
		CGMAvailable:    c.cgmAvailable,
		AlgorithmRate:   c.algorithmRate,
	}
}

// BasalCycleEnabled reports whether basal is delivered in 5-minute cycles.
func (ps *PumpState) BasalCycleEnabled() bool {
	c := ps.basalCycle
	c.mtx.Lock()
	defer c.mtx.Unlock()
	return c.enabled
}

// SetBasalCycleEnabled turns the cycle on or off. Turning it on writes the
// first 279 at once, so the cadence runs from that instant.
func (ps *PumpState) SetBasalCycleEnabled(enabled bool) {
	c := ps.basalCycle
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if enabled == c.enabled {
		return
	}
	c.enabled = enabled
	if !enabled {
		c.events = nil
		log.Info("Basal cycle off: basal is delivered continuously")
		return
	}

	now := ps.Now()
	minutes := int(ps.PumpTimeFor(now)%86400) / 60
	ps.mutex.RLock()
	c.pcm = ps.currentPCMUnlocked(c)
	c.segment = -1
	if len(ps.idpProfiles) > 0 && len(ps.idpProfiles[0].Segments) > 0 {
		c.segment = segmentInForce(ps.idpProfiles[0].Segments, minutes)
	}
	ps.mutex.RUnlock()
	c.crossedWhileSuspended = false
	c.through = now
	c.next = now
	log.Infof("Basal cycle on: a 279 every %v from %s", BasalCycleLength, now.Format(time.RFC3339))
	ps.advanceBasalCycleLocked(now, true)
}

// SetControlIQAlgorithmRate scripts the rate Control-IQ runs at each cycle, in
// U/hr. A negative rate runs the profile rate.
func (ps *PumpState) SetControlIQAlgorithmRate(rate float64) {
	ps.AdvanceBasalCycle(ps.Now())
	c := ps.basalCycle
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.algorithmRate = rate
}

// SetCGMAvailable sets whether Control-IQ has a CGM. Losing it moves Control-IQ
// to the profile at the next cycle, with a ControlIQPcmChange there.
func (ps *PumpState) SetCGMAvailable(available bool) {
	ps.AdvanceBasalCycle(ps.Now())
	c := ps.basalCycle
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.cgmAvailable = available
}

// RebaseBasalCycle keeps the cycle's phase across a clock that was set rather
// than advanced, so the jump writes no 279s.
func (ps *PumpState) RebaseBasalCycle() {
	c := ps.basalCycle
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if !c.enabled {
		return
	}
	now := ps.Now()
	remaining := c.next.Sub(c.through)
	if remaining < 0 || remaining > BasalCycleLength {
		remaining = 0
	}
	c.through = now
	c.next = now.Add(remaining)
	c.events = nil
}

// AdvanceBasalCycle processes everything the cycle has due up to until, in time
// order, and returns the temp rates it found expired on the way, for the caller
// to raise their events. It is a no-op while the cycle is off.
func (ps *PumpState) AdvanceBasalCycle(until time.Time) []TempRateSnapshot {
	c := ps.basalCycle
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if !c.enabled {
		return nil
	}
	return ps.advanceBasalCycleLocked(until, true)
}

// advanceBasalCycleTo is AdvanceBasalCycle for a caller about to end the temp
// rate itself, at until: it leaves the temp rate running for that caller.
func (ps *PumpState) advanceBasalCycleTo(until time.Time) {
	c := ps.basalCycle
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.enabled {
		ps.advanceBasalCycleLocked(until, false)
	}
}

func (ps *PumpState) advanceBasalCycleLocked(until time.Time, expireTemps bool) []TempRateSnapshot {
	c := ps.basalCycle
	var expired []TempRateSnapshot

	for {
		switch step, at := ps.nextBasalCycleStep(until, expireTemps); step {
		case stepTempExpiry:
			if ended, _, ok := ps.endTempRate(at); ok {
				expired = append(expired, ended)
			}
		case stepEvent:
			ps.applyBasalCycleEvent(c.events[0])
			c.events = c.events[1:]
		case stepBoundary:
			ps.writeBasalCycle(at)
		default:
			if until.After(c.through) {
				c.through = until
			}
			ps.followProfileAt(until)
			return expired
		}
	}
}

type basalCycleStep int

const (
	stepDone basalCycleStep = iota
	stepTempExpiry
	stepEvent
	stepBoundary
)

// nextBasalCycleStep is the earliest thing due by until. At one instant a temp
// rate ends first, so the 279 there sees it ended, and a pending growth or
// record goes before a new 279.
func (ps *PumpState) nextBasalCycleStep(until time.Time, expireTemps bool) (basalCycleStep, time.Time) {
	c := ps.basalCycle
	step, at := stepDone, until
	consider := func(candidate basalCycleStep, when time.Time) {
		if !when.After(until) && (step == stepDone || when.Before(at)) {
			step, at = candidate, when
		}
	}
	if expireTemps {
		if temp := ps.GetTempRate(); temp.Active && !temp.EndTime.IsZero() {
			consider(stepTempExpiry, temp.EndTime)
		}
	}
	if len(c.events) > 0 {
		consider(stepEvent, c.events[0].at)
	}
	consider(stepBoundary, c.next)
	return step, at
}

// writeBasalCycle writes the 279 at at and commits the cycle's insulin.
func (ps *PumpState) writeBasalCycle(at time.Time) {
	c := ps.basalCycle
	ps.followProfileAt(at)

	ps.mutex.RLock()
	suspended := ps.PumpingSuspended
	closedLoop := ps.ClosedLoopEnabled
	profileMilli := int(math.Round(ps.Basal.CurrentRate * 1000))
	tempActive := ps.Basal.TempBasalActive
	tempPercent := ps.Basal.TempBasalPercent
	ps.mutex.RUnlock()

	if closedLoop && !suspended {
		want := PCMClosedLoop
		if !c.cgmAvailable {
			want = PCMNoCGM
		}
		if c.pcm != want {
			ps.writePCMChange(at, want, suspended, closedLoop)
		}
	}

	source, rateMilli := BasalSourceProfile, profileMilli
	switch {
	case suspended:
		source, rateMilli = BasalSourceSuspended, 0
	case closedLoop && c.cgmAvailable:
		source = BasalSourceControlIQ
		if c.algorithmRate >= 0 {
			rateMilli = int(math.Round(c.algorithmRate * 1000))
		}
	case closedLoop:
		// No CGM: Control-IQ runs the profile.
	case tempActive:
		// Exactly the percent of the profile, not rounded to 0.01 U/hr.
		source, rateMilli = BasalSourceTemp, int(math.Round(float64(profileMilli)*float64(tempPercent)/100))
	}

	algorithmMilli, tempMilli := 0, 0
	switch source {
	case BasalSourceControlIQ:
		algorithmMilli = rateMilli
	case BasalSourceTemp:
		tempMilli = rateMilli
	}
	ps.AddHistoryLogEntryAtWithExtra(HistoryBasalDelivery, "BasalDelivery", at,
		map[string]interface{}{
			"commandedRateSource": source,
			"basalDeliveryFlags":  0,
			"commandedRate":       rateMilli,
			"profileBasalRate":    profileMilli,
			"algorithmRate":       algorithmMilli,
			"tempRate":            tempMilli,
		},
		map[string]interface{}{"cycleSeconds": BasalCycleLength.Seconds()})

	rate := float64(rateMilli) / 1000
	units := rate * BasalCycleLength.Hours()
	ps.mutex.Lock()
	ps.Reservoir.CurrentUnits = math.Max(0, ps.Reservoir.CurrentUnits-units)
	ps.IOB += units
	ps.TDD += units
	ps.mutex.Unlock()

	c.latest, c.rate, c.source = at, rate, source
	if suspended {
		c.crossedWhileSuspended = true
	}
	c.next = at.Add(BasalCycleLength)
	c.events = append(c.events,
		basalCycleEvent{at: at.Add(dailyBasalGrowthDelay), kind: cycleGrowth, units: units},
		basalCycleEvent{at: at.Add(dailyBasalRecordDelay), kind: cycleDailyRecord})
	sort.SliceStable(c.events, func(i, j int) bool { return c.events[i].at.Before(c.events[j].at) })
	if at.After(c.through) {
		c.through = at
	}
}

func (ps *PumpState) applyBasalCycleEvent(event basalCycleEvent) {
	c := ps.basalCycle
	ps.rollDailyTotal(event.at)
	switch event.kind {
	case cycleGrowth:
		c.dailyTotal += event.units
	case cycleDailyRecord:
		ps.mutex.RLock()
		iob := ps.IOB
		battery := ps.Battery.Percentage
		ps.mutex.RUnlock()
		ps.AddHistoryLogEntryAt(HistoryDailyBasal, "DailyBasal", event.at, map[string]interface{}{
			"dailyTotalBasal":  c.dailyTotal,
			"lastBasalRate":    c.rate,
			"iob":              iob,
			"finalEventForDay": false,
			"batteryChargeRaw": battery,
			"lipoMv":           3900,
		})
	}
	if event.at.After(c.through) {
		c.through = event.at
	}
}

// rollDailyTotal starts a new daily total at the pump's midnight.
func (ps *PumpState) rollDailyTotal(at time.Time) {
	c := ps.basalCycle
	day := int64(ps.PumpTimeFor(at)) / 86400
	if day != c.dailyDay {
		c.dailyDay = day
		c.dailyTotal = 0
	}
}

// followProfileAt puts the active profile's segment at the instant at in force
// when it differs from the last, rescaling a running temp rate with it, as a
// pump does at each segment boundary. Only the cycle does this: without it the
// emulator's profile rate moves only when a profile is written.
func (ps *PumpState) followProfileAt(at time.Time) {
	c := ps.basalCycle
	minutes := int(ps.PumpTimeFor(at)%86400) / 60
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	if len(ps.idpProfiles) == 0 || len(ps.idpProfiles[0].Segments) == 0 {
		return
	}
	segment := segmentInForce(ps.idpProfiles[0].Segments, minutes)
	if segment == c.segment {
		return
	}
	c.segment = segment
	ps.followActiveProfileUnlocked(minutes)
}

// currentPCMUnlocked is the PCM the pump's state calls for. The caller holds the
// pump state's mutex.
func (ps *PumpState) currentPCMUnlocked(c *basalCycle) int {
	switch {
	case ps.PumpingSuspended:
		return PCMNoControl
	case !ps.ClosedLoopEnabled:
		return PCMOpenLoop
	case c.cgmAvailable:
		return PCMClosedLoop
	default:
		return PCMNoCGM
	}
}

func (ps *PumpState) writePCMChange(at time.Time, pcm int, suspended, closedLoop bool) {
	c := ps.basalCycle
	ps.AddHistoryLogEntryAt(HistoryControlIQPcmChange, "ControlIQPcmChange", at, map[string]interface{}{
		"currentPcm":                 pcm,
		"previousPcm":                c.pcm,
		"pumpSuspended":              suspended,
		"calculationAvailable":       true,
		"cgmAvailable":               c.cgmAvailable,
		"closedLoopPreferred":        closedLoop,
		"sufficientClosedLoopParams": true,
	})
	c.pcm = pcm
}

// basalCycleSuspended records a stop for the cycle: the cycle under way is not
// taken back, later 279s say suspended, and PCM goes to no control.
func (ps *PumpState) basalCycleSuspended(at time.Time) {
	c := ps.basalCycle
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if !c.enabled {
		return
	}
	c.crossedWhileSuspended = false
	ps.mutex.RLock()
	closedLoop := ps.ClosedLoopEnabled
	ps.mutex.RUnlock()
	ps.writePCMChange(at, PCMNoControl, true, closedLoop)
}

// basalCycleResumed records a restart: after a suspend that spanned a cycle
// boundary the pump writes a 279 at the resume and its cadence restarts from
// it; within the suspend's own cycle it writes none and delivery restarts at
// the next cycle.
func (ps *PumpState) basalCycleResumed(at time.Time) {
	c := ps.basalCycle
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if !c.enabled {
		return
	}
	ps.mutex.RLock()
	pcm := ps.currentPCMUnlocked(c)
	closedLoop := ps.ClosedLoopEnabled
	ps.mutex.RUnlock()
	ps.writePCMChange(at, pcm, false, closedLoop)

	if c.crossedWhileSuspended {
		c.crossedWhileSuspended = false
		c.next = at
		ps.writeBasalCycle(at)
	}
}

// basalCycleClosedLoopChanged writes the PCM a Control-IQ switch moves to. A
// suspended pump stays at no control until its resume.
func (ps *PumpState) basalCycleClosedLoopChanged(at time.Time) {
	c := ps.basalCycle
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if !c.enabled {
		return
	}
	ps.mutex.RLock()
	suspended := ps.PumpingSuspended
	closedLoop := ps.ClosedLoopEnabled
	pcm := ps.currentPCMUnlocked(c)
	ps.mutex.RUnlock()
	if !suspended && pcm != c.pcm {
		ps.writePCMChange(at, pcm, false, closedLoop)
	}
}

// RecordSetTempRateResponse writes the SetTempRateResponse (309) record a pump
// keeps for every temp rate set over Bluetooth. A temp rate started on the
// pump's own screen has none. Only the basal cycle writes it, so a driver's
// existing view of the log does not change while the cycle is off.
func (ps *PumpState) RecordSetTempRateResponse(status, tempRateID int) {
	if !ps.BasalCycleEnabled() {
		return
	}
	ps.AddHistoryLogEntryWithExtra(HistorySetTempRateResponse, "SetTempRateResponse",
		map[string]interface{}{"status": status, "unknown11": 0, "tempRateId": tempRateID},
		nil)
}
