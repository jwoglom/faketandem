package state

import (
	"math"
	"testing"
	"time"
)

func assertUnits(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("%s = %.6f, want %.6f", name, got, want)
	}
}

// deliverBolus runs a bolus to completion within one second.
func deliverBolus(ps *PumpState, clock *ManualClock, sim *Simulator, units float64) {
	ps.SetBolusRate(units)
	ps.StartBolusWithSource(units, 1, BolusSourceBluetoothRemote, 0)
	advance(clock, sim, time.Second)
}

func TestIOBWithControlIQOffCountsBolusesAloneOverTheProfilesDuration(t *testing.T) {
	ps, clock, sim, t0 := cyclePump(t)
	deliverBolus(ps, clock, sim, 2.0)
	startTemp(ps, 1, 200, 2*time.Hour)

	advance(clock, sim, time.Hour-time.Second)
	iob := ps.ReadIOB()
	if iob.Type != IOBTypeMudaliar {
		t.Errorf("iobType = %d, want Mudaliar", iob.Type)
	}
	assertUnits(t, "mudaliar", iob.Mudaliar, 2.0*(1-1.0/5))
	assertUnits(t, "mudaliar total", iob.MudaliarTotal, 2.0)
	assertUnits(t, "displayed", iob.Displayed(), iob.Mudaliar)
	if iob.TimeRemaining != 4*time.Hour {
		t.Errorf("time remaining = %v, want 4h", iob.TimeRemaining)
	}
	if iob.Swan6Hr <= iob.Mudaliar {
		t.Errorf("swan %.3f, want the temp's extra basal on top of the bolus", iob.Swan6Hr)
	}

	clock.Set(t0.Add(5 * time.Hour))
	iob = ps.ReadIOB()
	assertUnits(t, "mudaliar after the duration", iob.Mudaliar, 0)
	assertUnits(t, "mudaliar total after the duration", iob.MudaliarTotal, 0)
	if iob.TimeRemaining != 0 {
		t.Errorf("time remaining after the duration = %v", iob.TimeRemaining)
	}
}

func TestIOBFollowsTheActiveProfilesInsulinDuration(t *testing.T) {
	ps, clock, sim, t0 := cyclePump(t)
	profiles := DefaultIDPProfiles()
	profiles[0].InsulinDuration = 120
	ps.SetIDPProfiles(profiles)
	deliverBolus(ps, clock, sim, 3.0)

	clock.Set(t0.Add(time.Hour))
	assertUnits(t, "after one of two hours", ps.ReadIOB().Mudaliar, 1.5)
	clock.Set(t0.Add(2 * time.Hour))
	assertUnits(t, "after two hours", ps.ReadIOB().Mudaliar, 0)
}

func TestUnderControlIQThePumpShowsSwanIOBWithBasalAboveTheProfile(t *testing.T) {
	ps, clock, sim, t0 := cyclePump(t)
	ps.SetControlIQAlgorithmRate(2.2)
	ps.SetClosedLoopEnabled(true)
	deliverBolus(ps, clock, sim, 1.0)

	clock.Set(t0.Add(5 * time.Minute))
	sim.Tick()
	iob := ps.ReadIOB()
	if iob.Type != IOBTypeSwan6Hr {
		t.Errorf("iobType = %d, want Swan 6hr", iob.Type)
	}
	assertUnits(t, "mudaliar", iob.Mudaliar, 1.0*(1-5.0/300))
	// The 279 at t0+5m runs 1.2 U/hr above the profile for five minutes.
	assertUnits(t, "swan", iob.Swan6Hr, 1.0*(1-5.0/360)+0.1)
	assertUnits(t, "displayed", iob.Displayed(), iob.Swan6Hr)
}

func TestSwanIOBNeverGoesBelowZero(t *testing.T) {
	ps, clock, sim, _ := cyclePump(t)
	ps.SetControlIQAlgorithmRate(0)
	ps.SetClosedLoopEnabled(true)
	advance(clock, sim, time.Hour)

	assertUnits(t, "swan", ps.ReadIOB().Swan6Hr, 0)
	assertUnits(t, "displayed", ps.GetIOB(), 0)
}

func TestAStoppedBolusCountsWhatItDelivered(t *testing.T) {
	ps, clock, sim, t0 := cyclePump(t)
	ps.SetBolusRate(0.05)
	ps.StartBolusWithSource(3.0, 1, BolusSourceBluetoothRemote, 0)
	advance(clock, sim, 20*time.Second)
	ps.EndBolus(BolusEndReasonStopped, nil)

	clock.Set(t0.Add(time.Hour))
	iob := ps.ReadIOB()
	assertUnits(t, "mudaliar", iob.Mudaliar, 1.0*(1-1.0/5))
	assertUnits(t, "mudaliar total", iob.MudaliarTotal, 1.0)
}

func TestABolusDeclaredCompleteCountsInFull(t *testing.T) {
	ps, clock, sim, _ := cyclePump(t)
	ps.SetBolusRate(0.05)
	ps.StartBolusWithSource(3.0, 1, BolusSourceBluetoothRemote, 0)
	advance(clock, sim, 20*time.Second)
	ps.EndBolus(BolusEndReasonCompleted, nil)

	assertUnits(t, "mudaliar total", ps.ReadIOB().MudaliarTotal, 3.0)
}

func TestStagedIOBReplacesWhatThePumpCountedAndWearsOff(t *testing.T) {
	ps, clock, sim, _ := cyclePump(t)
	deliverBolus(ps, clock, sim, 2.0)
	ps.SetIOB(0.5)
	assertUnits(t, "staged", ps.GetIOB(), 0.5)

	clock.Advance(150 * time.Minute)
	assertUnits(t, "half the duration later", ps.GetIOB(), 0.25)
}

func TestWithoutTheCycleATempCountsOnlyTowardSwanIOB(t *testing.T) {
	ps, clock := lifecyclePump(t)
	runTempRate(ps, 1, time.Hour)
	sim := NewSimulator(ps, time.Second)
	sim.Tick()
	for i := 0; i < 10; i++ {
		advance(clock, sim, time.Minute)
	}

	iob := ps.ReadIOB()
	assertUnits(t, "mudaliar", iob.Mudaliar, 0)
	// 0.5 U/hr above the profile for ten minutes, a few minutes into six hours.
	if want := 0.5 / 6; iob.Swan6Hr > want || iob.Swan6Hr < want*0.97 {
		t.Errorf("swan = %.4f, want just under %.4f", iob.Swan6Hr, want)
	}
}
