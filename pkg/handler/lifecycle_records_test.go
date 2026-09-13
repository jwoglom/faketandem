package handler

import (
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/state"
)

// newLifecycleRouter is newFieldsRouter plus the clock handle, so a test can
// put time between a temp rate starting and whatever ends it.
func newLifecycleRouter(t *testing.T) (*Router, *state.PumpState, *state.ManualClock) {
	t.Helper()

	ps := state.NewPumpState()
	clock := state.NewFrozenClock(time.Date(2024, time.March, 5, 12, 0, 0, 0, time.UTC))
	ps.SetClock(clock)

	router := NewRouter(nil, ps, &recordingTransport{}, protocol.NewTransactionManager(time.Second),
		"go", "", "jar", "", "java", "")
	return router, ps, clock
}

// recordsOfType returns every history record of one type, oldest first.
func recordsOfType(t *testing.T, ps *state.PumpState, typeName string) []state.HistoryLogEntry {
	t.Helper()

	_, last := ps.GetHistoryLogSequenceRange()
	var out []state.HistoryLogEntry
	for _, entry := range ps.GetHistoryLogEntries(0, last) {
		if entry.Type == typeName {
			out = append(out, entry)
		}
	}
	return out
}

func recordCount(t *testing.T, ps *state.PumpState, typeName string) int {
	t.Helper()
	return len(recordsOfType(t, ps, typeName))
}

// startTempRate applies the basal state a SetTempRateRequest produces.
func startTempRate(t *testing.T, router *Router, ps *state.PumpState, id int, percent int, rate float64, duration time.Duration) {
	t.Helper()

	start := ps.Now()
	router.applyStateChange(StateChange{Type: StateChangeBasal, Data: &state.BasalState{
		CurrentRate:      1.0,
		TempBasalActive:  true,
		TempBasalRate:    rate,
		TempBasalPercent: percent,
		TempBasalStart:   start,
		TempBasalEnd:     start.Add(duration),
		TempRateID:       id,
	}})
}

// stopTempRate applies the basal state a StopTempRateRequest produces: back to
// the profile rate with no temp rate at all.
func stopTempRate(t *testing.T, router *Router) {
	t.Helper()
	router.applyStateChange(StateChange{Type: StateChangeBasal, Data: &state.BasalState{CurrentRate: 1.0}})
}

// assertTempRateCompleted checks the one thing a driver's timeline needs from a
// TempRateCompleted: which temp rate it closes, when it closed, and how much of
// the programmed duration was still to run.
func assertTempRateCompleted(t *testing.T, entry state.HistoryLogEntry, tempRateID int, when time.Time, timeLeft int) {
	t.Helper()

	if got, want := len(entry.Data), 2; got != want {
		t.Errorf("TempRateCompleted has %d wire fields, want exactly %d (tempRateId and timeLeft are all the record format carries); has %v",
			got, want, entry.Data)
	}
	if got := entry.Data["tempRateId"]; got != tempRateID {
		t.Errorf("TempRateCompleted tempRateId = %v, want the %d it closed", got, tempRateID)
	}
	if got := entry.Data["timeLeft"]; got != timeLeft {
		t.Errorf("TempRateCompleted timeLeft = %v, want %d seconds", got, timeLeft)
	}
	if !entry.Timestamp.Equal(when) {
		t.Errorf("TempRateCompleted stamped at %v, want the true end instant %v", entry.Timestamp, when)
	}
	for _, name := range []string{"tempRate", "normalRate", "percent", "durationSeconds", "deliveredUnits", "endedEarly"} {
		if _, ok := entry.Extra[name]; !ok {
			t.Errorf("TempRateCompleted extra is missing %q; has %v", name, entry.Extra)
		}
	}
}

// TestDriverStopTempRateWritesOneCompletion is the gap the conformance run
// found: a temp rate ended by a DRIVER command wrote no TempRateCompleted at
// all, so a temp rate stopped over the protocol stayed open forever in a
// timeline built from the history log and no cancel scenario could close.
func TestDriverStopTempRateWritesOneCompletion(t *testing.T) {
	router, ps, clock := newLifecycleRouter(t)

	startTempRate(t, router, ps, 4, 150, 1.5, 30*time.Minute)
	clock.Advance(10 * time.Minute)
	stopInstant := ps.Now()

	stopTempRate(t, router)

	completed := recordsOfType(t, ps, "TempRateCompleted")
	if len(completed) != 1 {
		t.Fatalf("a driver-commanded stop wrote %d TempRateCompleted records, want exactly 1", len(completed))
	}
	// Twenty of the thirty programmed minutes were still to run.
	assertTempRateCompleted(t, completed[0], 4, stopInstant, 20*60)

	if got := completed[0].Extra["endedEarly"]; got != true {
		t.Errorf("extra endedEarly = %v, want true for a temp rate cut short", got)
	}
	if got, want := completed[0].Extra["deliveredUnits"], 1.5*600/3600; got != want {
		t.Errorf("extra deliveredUnits = %v, want %v (ten minutes at 1.5 U/hr)", got, want)
	}
	if ps.GetTempRate().Active {
		t.Error("the temp rate is still active after a stop")
	}

	// Applying the same stop again is not a second ending.
	stopTempRate(t, router)
	if got := recordCount(t, ps, "TempRateCompleted"); got != 1 {
		t.Errorf("a repeated stop wrote %d TempRateCompleted records in total, want 1", got)
	}
}

// TestDriverReplacementTempRateClosesThePreviousOne: a second SetTempRate while
// one is running replaces it, and the one it replaces ends at the replacement
// instant -- with exactly one completion record, written before the new rate's
// activation.
func TestDriverReplacementTempRateClosesThePreviousOne(t *testing.T) {
	router, ps, clock := newLifecycleRouter(t)

	startTempRate(t, router, ps, 1, 150, 1.5, 30*time.Minute)
	clock.Advance(5 * time.Minute)
	replacedAt := ps.Now()
	startTempRate(t, router, ps, 2, 50, 0.5, 60*time.Minute)

	completed := recordsOfType(t, ps, "TempRateCompleted")
	if len(completed) != 1 {
		t.Fatalf("replacing a temp rate wrote %d TempRateCompleted records, want exactly 1", len(completed))
	}
	assertTempRateCompleted(t, completed[0], 1, replacedAt, 25*60)

	// Order matters for a timeline: the old rate closes before the new one opens.
	activated := recordsOfType(t, ps, "TempRateActivated")
	if len(activated) != 2 {
		t.Fatalf("got %d TempRateActivated records, want 2", len(activated))
	}
	if completed[0].Sequence > activated[1].Sequence {
		t.Errorf("the replaced temp rate's completion (seq %d) was logged after the replacement's activation (seq %d)",
			completed[0].Sequence, activated[1].Sequence)
	}

	// And the replacement itself closes when it is stopped, naming its own id.
	clock.Advance(10 * time.Minute)
	stoppedAt := ps.Now()
	stopTempRate(t, router)

	completed = recordsOfType(t, ps, "TempRateCompleted")
	if len(completed) != 2 {
		t.Fatalf("got %d TempRateCompleted records after stopping the replacement, want 2", len(completed))
	}
	assertTempRateCompleted(t, completed[1], 2, stoppedAt, 50*60)
}

// TestDriverSuspendEndsEverythingItStops: a suspend commanded over the protocol
// is a real stop. It halts an in-progress bolus and any running temp rate, and
// each of those endings is its own record -- written before the
// PumpingSuspended record that caused them. This path used to set only the
// suspend flag.
func TestDriverSuspendEndsEverythingItStops(t *testing.T) {
	router, ps, clock := newLifecycleRouter(t)

	router.applyStateChange(StateChange{Type: StateChangeBolus, Data: &state.BolusState{
		Active:     true,
		BolusID:    9,
		UnitsTotal: 3.0,
		SourceID:   state.BolusSourceBluetoothRemote,
	}})
	startTempRate(t, router, ps, 7, 150, 1.5, 30*time.Minute)

	clock.Advance(2 * time.Minute)
	suspendedAt := ps.Now()
	router.applyStateChange(StateChange{Type: StateChangeSuspend, Data: true})

	bolusEnds := recordsOfType(t, ps, "BolusCompleted")
	if len(bolusEnds) != 1 {
		t.Fatalf("the suspend wrote %d BolusCompleted records, want exactly 1", len(bolusEnds))
	}
	if got := bolusEnds[0].Data["completionStatusId"]; got != state.BolusEndReasonStopped {
		t.Errorf("completionStatusId = %v, want stopped (%d)", got, state.BolusEndReasonStopped)
	}
	if got := bolusEnds[0].Data["bolusId"]; got != uint32(9) {
		t.Errorf("BolusCompleted bolusId = %v, want the 9 the suspend stopped", got)
	}

	tempEnds := recordsOfType(t, ps, "TempRateCompleted")
	if len(tempEnds) != 1 {
		t.Fatalf("the suspend wrote %d TempRateCompleted records, want exactly 1", len(tempEnds))
	}
	assertTempRateCompleted(t, tempEnds[0], 7, suspendedAt, 28*60)

	suspends := recordsOfType(t, ps, "PumpingSuspended")
	if len(suspends) != 1 {
		t.Fatalf("got %d PumpingSuspended records, want exactly 1", len(suspends))
	}
	if suspends[0].Sequence < bolusEnds[0].Sequence || suspends[0].Sequence < tempEnds[0].Sequence {
		t.Error("PumpingSuspended was logged before the endings it caused")
	}
	if ps.GetTempRate().Active || ps.IsBolusActive() {
		t.Error("delivery is suspended but the bolus or temp rate is still running")
	}

	// A second suspend command is not a second transition.
	router.applyStateChange(StateChange{Type: StateChangeSuspend, Data: true})
	if got := recordCount(t, ps, "PumpingSuspended"); got != 1 {
		t.Errorf("a repeated suspend wrote %d PumpingSuspended records in total, want 1", got)
	}

	router.applyStateChange(StateChange{Type: StateChangeSuspend, Data: false})
	if got := recordCount(t, ps, "PumpingResumed"); got != 1 {
		t.Fatalf("got %d PumpingResumed records, want exactly 1", got)
	}
	// ... and neither is a second resume.
	router.applyStateChange(StateChange{Type: StateChangeSuspend, Data: false})
	if got := recordCount(t, ps, "PumpingResumed"); got != 1 {
		t.Errorf("a repeated resume wrote %d PumpingResumed records in total, want 1", got)
	}
}

// TestDriverBolusCancelReportsTheRightEndReason: a canceled bolus is
// STOPPED_USER_TERMINATED (0), in both halves of its ending. The emulator used
// to report 2, which is STOPPED_MALFUNCTION -- so every canceled bolus looked
// to a driver like a pump malfunction.
func TestDriverBolusCancelReportsTheRightEndReason(t *testing.T) {
	router, ps, _ := newLifecycleRouter(t)

	router.applyStateChange(StateChange{Type: StateChangeBolus, Data: &state.BolusState{
		Active: true, BolusID: 12, UnitsTotal: 2.0, SourceID: state.BolusSourceBluetoothRemote,
	}})
	router.applyStateChange(StateChange{Type: StateChangeBolus, Data: &state.BolusState{Active: false}})

	completed := recordsOfType(t, ps, "BolusCompleted")
	if len(completed) != 1 {
		t.Fatalf("canceling wrote %d BolusCompleted records, want exactly 1", len(completed))
	}
	if got := completed[0].Data["completionStatusId"]; got != 0 {
		t.Errorf("completionStatusId = %v, want 0 (STOPPED_USER_TERMINATED)", got)
	}

	record, ok := ps.GetLastBolus()
	if !ok {
		t.Fatal("no last-bolus record after a cancel")
	}
	if record.EndReasonID != state.BolusEndReasonStopped {
		t.Errorf("last bolus endReasonId = %d, want %d", record.EndReasonID, state.BolusEndReasonStopped)
	}

	// Canceling again ends nothing: there is no bolus left to end.
	router.applyStateChange(StateChange{Type: StateChangeBolus, Data: &state.BolusState{Active: false}})
	if got := recordCount(t, ps, "BolusCompleted"); got != 1 {
		t.Errorf("a repeated cancel wrote %d BolusCompleted records in total, want 1", got)
	}
}
