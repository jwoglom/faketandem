package state

import (
	"testing"
	"time"
)

var testInstant = time.Date(2024, time.March, 5, 12, 0, 0, 0, time.UTC)

func TestManualClockFrozenStandsStill(t *testing.T) {
	c := NewFrozenClock(testInstant)

	first := c.Now()
	time.Sleep(5 * time.Millisecond)
	second := c.Now()

	if !first.Equal(testInstant) || !second.Equal(testInstant) {
		t.Fatalf("frozen clock moved: %v then %v, want %v", first, second, testInstant)
	}
	if !c.Frozen() {
		t.Error("Frozen() = false after Freeze(true)")
	}
}

func TestManualClockAdvance(t *testing.T) {
	c := NewFrozenClock(testInstant)

	c.Advance(90 * time.Second)
	if got, want := c.Now(), testInstant.Add(90*time.Second); !got.Equal(want) {
		t.Errorf("after Advance: Now() = %v, want %v", got, want)
	}

	c.Advance(-30 * time.Second)
	if got, want := c.Now(), testInstant.Add(60*time.Second); !got.Equal(want) {
		t.Errorf("after negative Advance: Now() = %v, want %v", got, want)
	}

	c.Set(testInstant)
	if got := c.Now(); !got.Equal(testInstant) {
		t.Errorf("after Set: Now() = %v, want %v", got, testInstant)
	}
}

func TestManualClockUnfrozenRunsAtWallSpeed(t *testing.T) {
	c := NewManualClock(testInstant)

	before := c.Now()
	time.Sleep(20 * time.Millisecond)
	after := c.Now()

	if !after.After(before) {
		t.Fatalf("unfrozen clock did not advance: %v then %v", before, after)
	}
	if elapsed := after.Sub(before); elapsed > time.Second {
		t.Errorf("unfrozen clock ran away: elapsed %v", elapsed)
	}
	// Unfreezing a frozen clock must not make time jump.
	c.Freeze(true)
	stopped := c.Now()
	c.Freeze(false)
	if resumed := c.Now(); resumed.Sub(stopped) > 50*time.Millisecond {
		t.Errorf("unfreezing jumped by %v", resumed.Sub(stopped))
	}
}

func TestPumpStateUsesInjectedClock(t *testing.T) {
	ps := NewPumpState()
	c := NewFrozenClock(testInstant)
	ps.SetClock(c)

	if got := ps.Now(); !got.Equal(testInstant) {
		t.Fatalf("PumpState.Now() = %v, want %v", got, testInstant)
	}

	// A bolus started on the manual clock is stamped with the manual clock.
	ps.StartBolus(3, 11)
	if got := ps.Bolus.StartTime; !got.Equal(testInstant) {
		t.Errorf("bolus StartTime = %v, want %v", got, testInstant)
	}

	// History entries too, both the wall-clock and the pump-epoch stamps.
	ps.AddHistoryLogEntry("TestEvent", nil)
	entries := ps.GetHistoryLogEntries(0, 1<<30)
	if len(entries) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(entries))
	}
	if !entries[0].Timestamp.Equal(testInstant) {
		t.Errorf("history Timestamp = %v, want %v", entries[0].Timestamp, testInstant)
	}
	if want := PumpTimeSeconds(testInstant); entries[0].PumpTime != want {
		t.Errorf("history PumpTime = %d, want %d", entries[0].PumpTime, want)
	}
}

func TestSetClockPreservesTimeSinceReset(t *testing.T) {
	ps := NewPumpState()
	ps.mutex.Lock()
	ps.TimeSinceReset = 300
	ps.mutex.Unlock()

	c := NewFrozenClock(testInstant)
	ps.SetClock(c)
	ps.UpdateTimeSinceReset()

	if got := ps.GetTimeSinceReset(); got != 300 {
		t.Errorf("TimeSinceReset = %d after clock swap, want it preserved at 300", got)
	}
}

func TestPumpClockOffsetSkewsEmittedTimestamps(t *testing.T) {
	ps := NewPumpState()
	c := NewFrozenClock(testInstant)
	ps.SetClock(c)

	if got, want := ps.PumpTimeNow(), PumpTimeSeconds(testInstant); got != want {
		t.Fatalf("PumpTimeNow() = %d, want %d with no offset", got, want)
	}

	ps.SetPumpClockOffset(8 * time.Second)

	if got := ps.GetPumpClockOffset(); got != 8*time.Second {
		t.Errorf("GetPumpClockOffset() = %v", got)
	}
	if got, want := ps.PumpTimeNow(), PumpTimeSeconds(testInstant.Add(8*time.Second)); got != want {
		t.Errorf("PumpTimeNow() = %d, want %d with an 8 s skew", got, want)
	}
	if got, want := ps.PumpTimeFor(testInstant), PumpTimeSeconds(testInstant)+8; got != want {
		t.Errorf("PumpTimeFor() = %d, want %d", got, want)
	}
	// The skew is a pump-clock lie, not a change to the emulator's own time:
	// internal durations must be unaffected.
	if got := ps.Now(); !got.Equal(testInstant) {
		t.Errorf("Now() = %v, want the unskewed %v", got, testInstant)
	}
}

func TestSimulatorTickUsesElapsedPumpTime(t *testing.T) {
	ps := NewPumpState()
	c := NewFrozenClock(testInstant)
	ps.SetClock(c)
	ps.SetBolusRate(0.1)

	sim := NewSimulator(ps, time.Second)
	sim.Tick() // establishes the baseline instant

	ps.StartBolus(2.0, 5)

	// 10 s of pump time at 0.1 U/s delivers 1 U, in a single tick and with no
	// wall-clock waiting at all.
	c.Advance(10 * time.Second)
	sim.Tick()

	ps.RLock()
	delivered := ps.Bolus.UnitsDelivered
	active := ps.Bolus.Active
	ps.RUnlock()

	if !active {
		t.Fatalf("bolus finished early: delivered %.3f", delivered)
	}
	if delivered < 0.99 || delivered > 1.01 {
		t.Errorf("delivered %.3f units after 10 s at 0.1 U/s, want ~1.0", delivered)
	}

	// Another 10 s finishes it and records the completion on the pump clock.
	c.Advance(10 * time.Second)
	sim.Tick()

	record, ok := ps.GetLastBolus()
	if !ok {
		t.Fatal("no last-bolus record after the bolus completed")
	}
	if want := testInstant.Add(20 * time.Second); !record.EndTime.Equal(want) {
		t.Errorf("last bolus EndTime = %v, want %v", record.EndTime, want)
	}
}

func TestSimulatorStalledBolusMakesNoProgress(t *testing.T) {
	ps := NewPumpState()
	c := NewFrozenClock(testInstant)
	ps.SetClock(c)

	sim := NewSimulator(ps, time.Second)
	sim.Tick()

	ps.StartBolus(5.0, 9)
	ps.mutex.Lock()
	ps.Bolus.Stalled = true
	ps.mutex.Unlock()

	c.Advance(time.Hour)
	sim.Tick()

	ps.RLock()
	delivered := ps.Bolus.UnitsDelivered
	active := ps.Bolus.Active
	ps.RUnlock()

	if delivered != 0 {
		t.Errorf("stalled bolus delivered %.3f units", delivered)
	}
	if !active {
		t.Error("stalled bolus should still be reported as in progress")
	}
}

func TestDefaultPumpStateUsesRealClock(t *testing.T) {
	ps := NewPumpState()
	if _, ok := ps.GetClock().(RealClock); !ok {
		t.Errorf("default clock is %T, want RealClock", ps.GetClock())
	}
	if got := ps.GetPumpClockOffset(); got != 0 {
		t.Errorf("default pump clock offset = %v, want 0", got)
	}
	if got := ps.GetBolusRate(); got != DefaultBolusRateUnitsPerSecond {
		t.Errorf("default bolus rate = %v", got)
	}
	if delta := time.Since(ps.Now()); delta > time.Second || delta < -time.Second {
		t.Errorf("default clock is %v away from wall time", delta)
	}
}
