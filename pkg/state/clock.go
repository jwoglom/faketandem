package state

import (
	"sync"
	"time"
)

// Clock is the single source of "now" for everything the emulator simulates:
// time-since-reset, bolus delivery progress, temp-rate expiry, history-log
// timestamps and every pump timestamp put on the wire.
//
// Production code uses RealClock. An integration harness swaps in ManualClock
// so a test can step the pump forward deterministically -- a bolus that takes
// 50 s of pump time finishes in one Advance rather than in 50 s of wall time,
// and a driver assertion about a dose's start or end second is exact rather
// than "within a tolerance".
type Clock interface {
	// Now returns the current time.
	Now() time.Time
}

// RealClock is the default Clock: the host wall clock.
type RealClock struct{}

// Now returns the host wall-clock time.
func (RealClock) Now() time.Time { return time.Now() }

// ManualClock is a Clock whose time is set and stepped explicitly.
//
// It has two modes. Unfrozen (the default) it runs at wall-clock speed from
// whatever instant it was last Set or Advanced to, so the pump keeps behaving
// like a pump while still starting from a chosen instant. Frozen, it stands
// completely still until Advance or Set moves it, which is what makes a test
// able to observe pump state strictly between two ticks.
//
// ManualClock is safe for concurrent use.
type ManualClock struct {
	mu sync.RWMutex
	// base is the manual time at the moment anchor was taken.
	base time.Time
	// anchor is the host time at which base was established. Unused while
	// frozen.
	anchor time.Time
	frozen bool
}

// NewManualClock returns a ManualClock reading now, running at wall-clock speed.
func NewManualClock(now time.Time) *ManualClock {
	return &ManualClock{base: now, anchor: time.Now()}
}

// NewFrozenClock returns a ManualClock stopped exactly at now. Time does not
// move until Advance, Set or Freeze(false) is called, which is what a test
// that wants to observe pump state between two ticks needs.
func NewFrozenClock(now time.Time) *ManualClock {
	return &ManualClock{base: now, anchor: time.Now(), frozen: true}
}

// Now returns the manual clock's current time.
func (c *ManualClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.nowLocked()
}

func (c *ManualClock) nowLocked() time.Time {
	if c.frozen {
		return c.base
	}
	return c.base.Add(time.Since(c.anchor))
}

// Set moves the clock to t, keeping its frozen/running mode.
func (c *ManualClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.base = t
	c.anchor = time.Now()
}

// Advance moves the clock forward by d (a negative d moves it back).
func (c *ManualClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.base = c.nowLocked().Add(d)
	c.anchor = time.Now()
}

// Freeze stops the clock where it is (frozen=true) or lets it run at
// wall-clock speed again from that instant (frozen=false).
func (c *ManualClock) Freeze(frozen bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Capture the current reading before switching modes, so neither
	// transition makes time jump.
	now := c.nowLocked()
	c.base = now
	c.anchor = time.Now()
	c.frozen = frozen
}

// Frozen reports whether the clock is currently stopped.
func (c *ManualClock) Frozen() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.frozen
}

// DefaultBolusRateUnitsPerSecond is the simulated bolus delivery speed used
// when nothing overrides it.
const DefaultBolusRateUnitsPerSecond = 0.05

// SetClock replaces the clock this state derives every "now" from.
//
// TimeSinceReset is preserved across the swap by re-basing StartTime onto the
// new clock: a pump that had been running for 300 s is still 300 s into its
// run after a test moves the clock to an arbitrary instant, which is what a
// driver reading TimeSinceResetResponse expects.
func (ps *PumpState) SetClock(clock Clock) {
	if clock == nil {
		clock = RealClock{}
	}

	ps.clockMtx.Lock()
	ps.clock = clock
	ps.clockMtx.Unlock()

	ps.mutex.Lock()
	elapsed := time.Duration(ps.TimeSinceReset) * time.Second
	now := clock.Now()
	ps.StartTime = now.Add(-elapsed)
	ps.CurrentTime = now
	ps.mutex.Unlock()
}

// GetClock returns the clock in use.
func (ps *PumpState) GetClock() Clock {
	ps.clockMtx.RLock()
	defer ps.clockMtx.RUnlock()
	if ps.clock == nil {
		return RealClock{}
	}
	return ps.clock
}

// Now returns the current time according to this pump's clock. It takes only
// the clock mutex, so it is safe to call while the main state mutex is held.
func (ps *PumpState) Now() time.Time {
	return ps.GetClock().Now()
}

// SetPumpClockOffset skews the pump's own clock relative to the Clock's time.
// A positive offset makes the pump report timestamps in the future.
func (ps *PumpState) SetPumpClockOffset(offset time.Duration) {
	ps.clockMtx.Lock()
	defer ps.clockMtx.Unlock()
	ps.pumpClockOffset = offset
}

// GetPumpClockOffset returns the pump-clock skew.
func (ps *PumpState) GetPumpClockOffset() time.Duration {
	ps.clockMtx.RLock()
	defer ps.clockMtx.RUnlock()
	return ps.pumpClockOffset
}

// PumpNow returns what the pump believes the current wall-clock time to be:
// the Clock's time plus the pump-clock skew.
func (ps *PumpState) PumpNow() time.Time {
	return ps.Now().Add(ps.GetPumpClockOffset())
}

// PumpTimeFor converts an instant on this pump's Clock into the pump-epoch
// seconds the wire carries, applying the pump-clock skew. Every emitted
// timestamp (bolus start and end, temp-rate start, history pumpTimeSec,
// TimeSinceResetResponse.currentTime) goes through here, so one offset moves
// all of them together.
func (ps *PumpState) PumpTimeFor(t time.Time) uint32 {
	return PumpTimeSeconds(t.Add(ps.GetPumpClockOffset()))
}

// GetBolusRate returns the simulated bolus delivery speed in units/second.
func (ps *PumpState) GetBolusRate() float64 {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	if ps.BolusRateUnitsPerSecond <= 0 {
		return DefaultBolusRateUnitsPerSecond
	}
	return ps.BolusRateUnitsPerSecond
}

// SetBolusRate sets the simulated bolus delivery speed in units/second. A
// non-positive rate restores the default.
func (ps *PumpState) SetBolusRate(unitsPerSecond float64) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	if unitsPerSecond <= 0 {
		unitsPerSecond = DefaultBolusRateUnitsPerSecond
	}
	ps.BolusRateUnitsPerSecond = unitsPerSecond
}
