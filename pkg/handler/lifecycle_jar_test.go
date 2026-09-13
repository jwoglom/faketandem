package handler

import (
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"
)

// TestDriverTempRateCommandsCloseTheirRecords drives the real messages a driver
// sends -- SetTempRateRequest, a second SetTempRateRequest, StopTempRateRequest
// and SuspendPumpingRequest -- through the real handlers and the router that
// applies what they return, and checks the history log a driver would then read
// back. This is the full path the conformance run exercised: the state changes
// the handlers produce are what the router turns into records, and a unit test
// that calls applyStateChange directly cannot catch a handler that stops
// producing them.
//
// Skipped unless FAKETANDEM_TEST_CLIPARSER_JAR is set.
func TestDriverTempRateCommandsCloseTheirRecords(t *testing.T) {
	bridge := testBridge(t)

	ps := state.NewPumpState()
	clock := state.NewFrozenClock(time.Date(2024, time.March, 5, 12, 0, 0, 0, time.UTC))
	ps.SetClock(clock)
	router := NewRouter(bridge, ps, &recordingTransport{}, protocol.NewTransactionManager(time.Second),
		"go", "", "jar", "", "java", "")

	apply := func(resp *Response) {
		t.Helper()
		for _, change := range resp.StateChanges {
			router.applyStateChange(change)
		}
	}
	handle := func(h MessageHandler, msg *pumpx2.ParsedMessage) *Response {
		t.Helper()
		resp, err := h.HandleMessage(msg, ps)
		if err != nil {
			t.Fatalf("%s handler failed: %v", msg.MessageType, err)
		}
		apply(resp)
		return resp
	}

	setTempRate := func(percent, minutes int) int {
		t.Helper()
		msg := roundTrip(t, bridge, bluetooth.CharControl, "SetTempRateRequest", map[string]interface{}{
			"minutes": minutes,
			"percent": percent,
		})
		resp := handle(NewSetTempRateHandler(bridge), msg)
		basal, ok := resp.StateChanges[0].Data.(*state.BasalState)
		if !ok {
			t.Fatalf("SetTempRateRequest produced a %T, want a *state.BasalState", resp.StateChanges[0].Data)
		}
		return basal.TempRateID
	}

	firstID := setTempRate(150, 30)

	// A second temp rate replaces the first, which ends here.
	clock.Advance(5 * time.Minute)
	replacedAt := ps.Now()
	secondID := setTempRate(50, 60)

	completed := recordsOfType(t, ps, "TempRateCompleted")
	if len(completed) != 1 {
		t.Fatalf("a replacing SetTempRateRequest wrote %d TempRateCompleted records, want exactly 1",
			len(completed))
	}
	assertTempRateCompleted(t, completed[0], firstID, replacedAt, 25*60)

	// StopTempRateRequest: the gap the conformance run found. This wrote no
	// completion at all, so a driver-stopped temp rate never closed.
	clock.Advance(10 * time.Minute)
	stoppedAt := ps.Now()
	stopMsg := roundTrip(t, bridge, bluetooth.CharControl, "StopTempRateRequest", nil)
	handle(NewStopTempRateHandler(bridge), stopMsg)

	completed = recordsOfType(t, ps, "TempRateCompleted")
	if len(completed) != 2 {
		t.Fatalf("StopTempRateRequest brought the total to %d TempRateCompleted records, want 2",
			len(completed))
	}
	assertTempRateCompleted(t, completed[1], secondID, stoppedAt, 50*60)
	if ps.GetTempRate().Active {
		t.Error("the temp rate is still active after StopTempRateRequest")
	}

	// A suspend commanded over the protocol ends a running temp rate too.
	thirdID := setTempRate(200, 45)
	clock.Advance(15 * time.Minute)
	suspendedAt := ps.Now()
	suspendMsg := roundTrip(t, bridge, bluetooth.CharControl, "SuspendPumpingRequest", nil)
	handle(NewSuspendPumpingHandler(bridge), suspendMsg)

	completed = recordsOfType(t, ps, "TempRateCompleted")
	if len(completed) != 3 {
		t.Fatalf("SuspendPumpingRequest brought the total to %d TempRateCompleted records, want 3",
			len(completed))
	}
	assertTempRateCompleted(t, completed[2], thirdID, suspendedAt, 30*60)
	if got := recordCount(t, ps, "PumpingSuspended"); got != 1 {
		t.Errorf("got %d PumpingSuspended records, want exactly 1", got)
	}

	resumeMsg := roundTrip(t, bridge, bluetooth.CharControl, "ResumePumpingRequest", nil)
	handle(NewResumePumpingHandler(bridge), resumeMsg)
	if got := recordCount(t, ps, "PumpingResumed"); got != 1 {
		t.Errorf("got %d PumpingResumed records, want exactly 1", got)
	}
}
