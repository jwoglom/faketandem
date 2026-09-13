package state

import (
	"sync"
	"testing"
	"time"
)

func lifecyclePump(t *testing.T) (*PumpState, *ManualClock) {
	t.Helper()

	ps := NewPumpState()
	clock := NewFrozenClock(time.Date(2024, time.March, 5, 12, 0, 0, 0, time.UTC))
	ps.SetClock(clock)
	return ps, clock
}

func countRecords(ps *PumpState, typeName string) int {
	_, last := ps.GetHistoryLogSequenceRange()
	n := 0
	for _, entry := range ps.GetHistoryLogEntries(0, last) {
		if entry.Type == typeName {
			n++
		}
	}
	return n
}

func runTempRate(ps *PumpState, id int, duration time.Duration) {
	start := ps.Now()
	ps.SetBasalState(&BasalState{
		CurrentRate:      1.0,
		TempBasalActive:  true,
		TempBasalRate:    1.5,
		TempBasalPercent: 150,
		TempBasalStart:   start,
		TempBasalEnd:     start.Add(duration),
		TempRateID:       id,
	})
}

// TestBolusEndReasonIDsMatchPumpX2 pins the wire values against pumpX2's
// LastBolusStatusAbstractResponse.BolusStatus and TandemKit's Swift enum, both
// of which number the members 0..6 with STOPPED_USER_TERMINATED first. The
// emulator used to call a stopped bolus 2, which is STOPPED_MALFUNCTION.
func TestBolusEndReasonIDsMatchPumpX2(t *testing.T) {
	for _, tc := range []struct {
		name string
		got  int
		want int
	}{
		{"STOPPED_USER_TERMINATED", BolusEndReasonStopped, 0},
		{"STOPPED_ALARM", BolusEndReasonStoppedAlarm, 1},
		{"STOPPED_MALFUNCTION", BolusEndReasonStoppedMalfunction, 2},
		{"COMPLETE", BolusEndReasonCompleted, 3},
		{"STOPPED_WIRELESS", BolusEndReasonStoppedWireless, 4},
		{"REJECTED_WIRELESS", BolusEndReasonRejectedWireless, 5},
		{"TERMINATED_PLGS", BolusEndReasonTerminatedPLGS, 6},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %d, want %d", tc.name, tc.got, tc.want)
		}
	}
}

// TestEndTempRateWritesOneRecordWhenPathsRace: the simulator expiring a temp
// rate on a tick and a driver stopping it at the same instant are two paths
// into the same ending. Exactly one of them may write the record.
func TestEndTempRateWritesOneRecordWhenPathsRace(t *testing.T) {
	ps, _ := lifecyclePump(t)
	runTempRate(ps, 3, 30*time.Minute)

	var wg sync.WaitGroup
	ended := make(chan bool, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, ok := ps.EndTempRate(time.Time{})
			ended <- ok
		}()
	}
	wg.Wait()
	close(ended)

	winners := 0
	for ok := range ended {
		if ok {
			winners++
		}
	}
	if winners != 1 {
		t.Errorf("%d callers claimed the same temp rate ending, want exactly 1", winners)
	}
	if got := countRecords(ps, "TempRateCompleted"); got != 1 {
		t.Errorf("%d TempRateCompleted records for one ending, want exactly 1", got)
	}
}

// TestEndBolusWritesOneRecordWhenPathsRace is the same pin for a bolus: a
// driver's cancel and the simulator's completion can land together.
func TestEndBolusWritesOneRecordWhenPathsRace(t *testing.T) {
	ps, _ := lifecyclePump(t)
	ps.StartBolusWithSource(2.0, 5, BolusSourceBluetoothRemote, 1)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ps.EndBolus(BolusEndReasonStopped, nil)
		}()
	}
	wg.Wait()

	if got := countRecords(ps, "BolusCompleted"); got != 1 {
		t.Errorf("%d BolusCompleted records for one bolus, want exactly 1", got)
	}
}

// TestSimulatorExpiryStampsTheProgrammedEnd keeps the path that already worked
// working: a temp rate that runs out on its own is recorded once, at the second
// it was programmed to end rather than at the tick that noticed, and with no
// time left to run.
func TestSimulatorExpiryStampsTheProgrammedEnd(t *testing.T) {
	ps, clock := lifecyclePump(t)
	sim := NewSimulator(ps, time.Second)
	sim.Tick()

	runTempRate(ps, 2, 30*time.Minute)
	programmedEnd := ps.GetTempRate().EndTime

	// A coarse tick: the clock lands well past the programmed end.
	clock.Advance(45 * time.Minute)
	sim.Tick()

	entries := []HistoryLogEntry{}
	_, last := ps.GetHistoryLogSequenceRange()
	for _, entry := range ps.GetHistoryLogEntries(0, last) {
		if entry.Type == "TempRateCompleted" {
			entries = append(entries, entry)
		}
	}
	if len(entries) != 1 {
		t.Fatalf("expiry wrote %d TempRateCompleted records, want exactly 1", len(entries))
	}
	if !entries[0].Timestamp.Equal(programmedEnd) {
		t.Errorf("expiry stamped at %v, want the programmed end %v", entries[0].Timestamp, programmedEnd)
	}
	if got := entries[0].Data["timeLeft"]; got != 0 {
		t.Errorf("timeLeft = %v, want 0 for a temp rate that ran its course", got)
	}
	if got := entries[0].Data["tempRateId"]; got != 2 {
		t.Errorf("tempRateId = %v, want 2", got)
	}
	if got := entries[0].Extra["endedEarly"]; got != false {
		t.Errorf("extra endedEarly = %v, want false", got)
	}

	// Further ticks have nothing left to end.
	clock.Advance(10 * time.Minute)
	sim.Tick()
	if got := countRecords(ps, "TempRateCompleted"); got != 1 {
		t.Errorf("later ticks brought the total to %d TempRateCompleted records, want 1", got)
	}
}

// TestSimulatorDoesNotReExpireAStoppedTempRate: a temp rate stopped before its
// programmed end is already closed, and the tick that passes that end must not
// write a second completion for it.
func TestSimulatorDoesNotReExpireAStoppedTempRate(t *testing.T) {
	ps, clock := lifecyclePump(t)
	sim := NewSimulator(ps, time.Second)
	sim.Tick()

	runTempRate(ps, 1, 30*time.Minute)
	clock.Advance(5 * time.Minute)
	stoppedAt := ps.Now()
	if _, _, ok := ps.EndTempRate(stoppedAt); !ok {
		t.Fatal("EndTempRate found no temp rate to end")
	}

	clock.Advance(40 * time.Minute)
	sim.Tick()

	if got := countRecords(ps, "TempRateCompleted"); got != 1 {
		t.Errorf("%d TempRateCompleted records after a stop and a later tick, want 1", got)
	}
}

// TestSuspendDeliveryIsOneTransition: stopping delivery ends what is running,
// writes one record for each ending and one for the stop, and a second suspend
// writes nothing at all.
func TestSuspendDeliveryIsOneTransition(t *testing.T) {
	ps, clock := lifecyclePump(t)

	ps.StartBolusWithSource(3.0, 11, BolusSourceBluetoothRemote, 1)
	runTempRate(ps, 6, 60*time.Minute)
	clock.Advance(time.Minute)
	suspendedAt := ps.Now()

	outcome := ps.SuspendDelivery("occlusion")
	if !outcome.Changed {
		t.Fatal("SuspendDelivery reported no change on a running pump")
	}
	if outcome.Bolus == nil || outcome.Bolus.BolusID != 11 {
		t.Errorf("outcome bolus = %+v, want the bolus 11 it stopped", outcome.Bolus)
	}
	if outcome.TempRate == nil || outcome.TempRate.TempRateID != 6 {
		t.Errorf("outcome temp rate = %+v, want the temp rate 6 it ended", outcome.TempRate)
	}
	for _, typeName := range []string{"BolusCompleted", "TempRateCompleted", "PumpingSuspended"} {
		if got := countRecords(ps, typeName); got != 1 {
			t.Errorf("%d %s records, want exactly 1", got, typeName)
		}
	}

	assertSuspendRecords(t, ps, suspendedAt)

	if outcome := ps.SuspendDelivery("user"); outcome.Changed {
		t.Error("suspending an already-suspended pump reported a transition")
	}
	if got := countRecords(ps, "PumpingSuspended"); got != 1 {
		t.Errorf("a second suspend brought the total to %d PumpingSuspended records, want 1", got)
	}

	if !ps.ResumeDelivery() {
		t.Fatal("ResumeDelivery reported no change on a suspended pump")
	}
	if ps.ResumeDelivery() {
		t.Error("resuming a running pump reported a transition")
	}
	if got := countRecords(ps, "PumpingResumed"); got != 1 {
		t.Errorf("%d PumpingResumed records, want exactly 1", got)
	}
}

// assertSuspendRecords checks the two records a suspend's own log entries have
// to get right: the temp rate it ended is stamped at the suspend instant, and
// the stop names the reason that caused it.
func assertSuspendRecords(t *testing.T, ps *PumpState, suspendedAt time.Time) {
	t.Helper()

	_, last := ps.GetHistoryLogSequenceRange()
	for _, entry := range ps.GetHistoryLogEntries(0, last) {
		switch entry.Type {
		case "TempRateCompleted":
			if !entry.Timestamp.Equal(suspendedAt) {
				t.Errorf("the suspend's TempRateCompleted is stamped %v, want the suspend instant %v",
					entry.Timestamp, suspendedAt)
			}
		case "PumpingSuspended":
			if entry.Data["reasonId"] != SuspendReasonIDMalfunction {
				t.Errorf("an occlusion suspend's reasonId = %v, want %d",
					entry.Data["reasonId"], SuspendReasonIDMalfunction)
			}
		}
	}
}
