package handler

import (
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/state"
)

// newFieldsRouter builds a router over a frozen pump with no bridge and no
// central: applyStateChange touches neither.
func newFieldsRouter(t *testing.T) (*Router, *state.PumpState) {
	t.Helper()

	ps := state.NewPumpState()
	ps.SetClock(state.NewFrozenClock(time.Date(2024, time.March, 5, 12, 0, 0, 0, time.UTC)))

	router := NewRouter(nil, ps, &recordingTransport{}, protocol.NewTransactionManager(time.Second),
		"go", "", "jar", "", "java", "")
	return router, ps
}

func lastRecordOfType(t *testing.T, ps *state.PumpState, typeName string) state.HistoryLogEntry {
	t.Helper()

	_, last := ps.GetHistoryLogSequenceRange()
	entries := ps.GetHistoryLogEntries(0, last)
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Type == typeName {
			return entries[i]
		}
	}
	t.Fatalf("no %s record in the log", typeName)
	return state.HistoryLogEntry{}
}

func hasFields(t *testing.T, entry state.HistoryLogEntry, names ...string) {
	t.Helper()
	for _, name := range names {
		if _, ok := entry.Data[name]; !ok {
			t.Errorf("%s record is missing wire field %q; has %v", entry.Type, name, entry.Data)
		}
	}
}

// TestProtocolPathWritesPumpX2FieldNames pins the protocol side of gap 3: a
// record written when the driver commands something must carry the same fields,
// under the same names, as the identical record written by a harness action.
// The two paths used to disagree -- bolusSize versus units, durationMilliseconds
// versus minutes, reasonId versus reason -- so a consumer had to know which one
// had produced a record before it could read it.
func TestProtocolPathWritesPumpX2FieldNames(t *testing.T) {
	router, ps := newFieldsRouter(t)

	router.applyStateChange(StateChange{Type: StateChangeBolus, Data: &state.BolusState{
		Active:     true,
		BolusID:    77,
		UnitsTotal: 2.5,
		SourceID:   state.BolusSourceBluetoothRemote,
	}})
	activated := lastRecordOfType(t, ps, "BolusActivated")
	hasFields(t, activated, "bolusId", "selectedIob", "iob", "bolusSize")
	if got := activated.Data["bolusSize"]; got != 2.5 {
		t.Errorf("bolusSize = %v, want 2.5", got)
	}
	if got := activated.Extra["bolusSourceId"]; got != state.BolusSourceBluetoothRemote {
		t.Errorf("extra bolusSourceId = %v, want %d", got, state.BolusSourceBluetoothRemote)
	}

	// Canceling over the protocol closes the bolus in the log too. This path
	// used to write the last-bolus record and no history record at all, so a
	// canceled bolus had a start in the log and no end.
	router.applyStateChange(StateChange{Type: StateChangeBolus, Data: &state.BolusState{Active: false}})
	completed := lastRecordOfType(t, ps, "BolusCompleted")
	hasFields(t, completed, "completionStatusId", "bolusId", "iob", "insulinDelivered", "insulinRequested")
	if got := completed.Data["completionStatusId"]; got != state.BolusEndReasonStopped {
		t.Errorf("completionStatusId = %v, want %d", got, state.BolusEndReasonStopped)
	}

	start := ps.Now()
	router.applyStateChange(StateChange{Type: StateChangeBasal, Data: &state.BasalState{
		CurrentRate:      1.0,
		TempBasalActive:  true,
		TempBasalRate:    1.5,
		TempBasalPercent: 150,
		TempBasalStart:   start,
		TempBasalEnd:     start.Add(30 * time.Minute),
		TempRateID:       4,
	}})
	tempOn := lastRecordOfType(t, ps, "TempRateActivated")
	hasFields(t, tempOn, "percent", "durationMilliseconds", "tempRateId")
	if got := tempOn.Data["durationMilliseconds"]; got != float64(30*60*1000) {
		t.Errorf("durationMilliseconds = %v, want %d", got, 30*60*1000)
	}

	// Ending it writes the completion record, naming the temp rate it closes.
	router.applyStateChange(StateChange{Type: StateChangeBasal, Data: &state.BasalState{CurrentRate: 1.0}})
	tempOff := lastRecordOfType(t, ps, "TempRateCompleted")
	hasFields(t, tempOff, "tempRateId", "timeLeft")
	if got, want := len(tempOff.Data), 2; got != want {
		t.Errorf("TempRateCompleted has %d wire fields, want exactly %d", got, want)
	}
	if got := tempOff.Data["tempRateId"]; got != 4 {
		t.Errorf("tempRateId = %v, want the 4 it closed", got)
	}
	if _, ok := tempOff.Extra["deliveredUnits"]; !ok {
		t.Error("TempRateCompleted extra is missing deliveredUnits")
	}

	router.applyStateChange(StateChange{Type: StateChangeSuspend, Data: true})
	suspended := lastRecordOfType(t, ps, "PumpingSuspended")
	hasFields(t, suspended, "preSuspendState", "insulinAmount", "reasonId", "rpaTimeout")
	if got := suspended.Data["reasonId"]; got != state.SuspendReasonIDUserAborted {
		t.Errorf("a protocol-commanded suspend's reasonId = %v, want %d",
			got, state.SuspendReasonIDUserAborted)
	}

	router.applyStateChange(StateChange{Type: StateChangeSuspend, Data: false})
	hasFields(t, lastRecordOfType(t, ps, "PumpingResumed"), "preResumeState", "insulinAmount")
}

// TestProtocolPathStampsTrueInstants pins gap 2 on the protocol side: a live
// record's Timestamp is the pump's own clock reading, and its PumpTime is that
// instant with the skew and the pump's zone applied -- not the other way round.
func TestProtocolPathStampsTrueInstants(t *testing.T) {
	router, ps := newFieldsRouter(t)
	now := ps.Now()
	ps.SetPumpClockOffset(90 * time.Second)

	router.applyStateChange(StateChange{Type: StateChangeSuspend, Data: true})
	entry := lastRecordOfType(t, ps, "PumpingSuspended")

	if !entry.Timestamp.Equal(now) {
		t.Errorf("Timestamp = %v, want the unskewed %v", entry.Timestamp, now)
	}
	if want := ps.PumpTimeFor(now); entry.PumpTime != want {
		t.Errorf("PumpTime = %d, want %d (skew and zone applied)", entry.PumpTime, want)
	}
	if unskewed := ps.PumpTimeFor(now) - 90; entry.PumpTime == unskewed {
		t.Error("PumpTime did not pick up the pump-clock skew")
	}
}
