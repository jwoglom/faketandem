package harness

import (
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/handler"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"
)

// These tests take each harness action and then read the pump the way a driver
// does -- through the real response messages, encoded and decoded by the
// cliparser jar. A pump-side action that does not move CurrentBasalStatus,
// CurrentBolusStatus, LastBolusStatus, TempRate and HistoryLogStatus is
// invisible to the driver under test, and a state change nobody can observe is
// not worth having.
//
// Skipped unless FAKETANDEM_TEST_CLIPARSER_JAR is set.

func testBridge(t *testing.T) *pumpx2.Bridge {
	t.Helper()

	jarPath := os.Getenv("FAKETANDEM_TEST_CLIPARSER_JAR")
	if jarPath == "" {
		t.Skip("FAKETANDEM_TEST_CLIPARSER_JAR not set, skipping real jar integration test")
	}

	bridge, err := pumpx2.NewBridge("", "jar", "", "java", jarPath)
	if err != nil {
		t.Fatalf("failed to create bridge: %v", err)
	}
	return bridge
}

// readBack runs a handler against the pump state and parses its response back
// into what a driver would decode.
func readBack(t *testing.T, bridge *pumpx2.Bridge, h handler.MessageHandler, ps *state.PumpState) *pumpx2.ParsedMessage {
	t.Helper()

	req := &pumpx2.ParsedMessage{TxID: 7, MessageType: h.MessageType(), Cargo: map[string]interface{}{}}
	resp, err := h.HandleMessage(req, ps)
	if err != nil {
		t.Fatalf("%s handler failed: %v", h.MessageType(), err)
	}
	if resp == nil || resp.ResponseMessage == nil || len(resp.ResponseMessage.Packets) == 0 {
		t.Fatalf("%s handler produced no response packets", h.MessageType())
	}

	parsed, err := bridge.ParseMessage(bluetooth.CharCurrentStatus, resp.ResponseMessage.Packets)
	if err != nil {
		t.Fatalf("failed to parse the %s response back: %v", h.MessageType(), err)
	}
	return parsed
}

// cargoInt reads a numeric cargo field, coercing the representations
// cliparser output can produce.
func cargoInt(t *testing.T, msg *pumpx2.ParsedMessage, key string) int64 {
	t.Helper()

	raw, ok := msg.Cargo[key]
	if !ok {
		t.Fatalf("%s has no cargo field %q (cargo=%v)", msg.MessageType, key, msg.Cargo)
	}
	switch v := raw.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			t.Fatalf("%s cargo %q = %q, not a number", msg.MessageType, key, v)
		}
		return parsed
	default:
		t.Fatalf("%s cargo %q has type %T", msg.MessageType, key, raw)
		return 0
	}
}

func assertCargo(t *testing.T, msg *pumpx2.ParsedMessage, key string, want int64) {
	t.Helper()

	if got := cargoInt(t, msg, key); got != want {
		t.Errorf("%s.%s = %d, want %d", msg.MessageType, key, got, want)
	}
}

// TestSuspendActionIsVisibleToTheDriver covers the whole pump-initiated stop
// as a driver sees it: basal reports suspended at a zero rate, the temp rate
// is gone, the bolus is finished with its partial volume, and the history log
// has grown.
func TestSuspendActionIsVisibleToTheDriver(t *testing.T) {
	bridge := testBridge(t)
	_, ps, _, mux := testHarness(t)

	mustDo(t, mux, http.MethodPut, "/api/clock", `{"mode":"manual","frozen":true,"now":"2024-03-05T12:00:00Z"}`)
	mustDo(t, mux, http.MethodPut, "/api/state", `{"basal_rate":1.0}`)
	mustDo(t, mux, http.MethodPost, "/api/state/tempbasal/start", `{"percent":150,"duration_minutes":30}`)
	body := mustDo(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":5,"rate":0.1}`)
	bolusID := int64(body["bolus_id"].(float64))
	mustDo(t, mux, http.MethodPost, "/api/clock/advance", `{"seconds":10}`)

	// Before the suspend the driver sees a temp rate running and a bolus in
	// progress.
	basal := readBack(t, bridge, handler.NewCurrentBasalStatusHandler(bridge), ps)
	assertCargo(t, basal, "basalModifiedBitmask", 0x02)
	assertCargo(t, basal, "currentBasalRate", 1500)
	assertCargo(t, basal, "profileBasalRate", 1000)

	bolus := readBack(t, bridge, handler.NewCurrentBolusStatusHandler(bridge), ps)
	assertCargo(t, bolus, "statusId", 1)
	assertCargo(t, bolus, "bolusId", bolusID)
	assertCargo(t, bolus, "timestamp", int64(ps.PumpTimeFor(testInstant)))

	temp := readBack(t, bridge, handler.NewTempRateHandler(bridge), ps)
	if got := cargoInt(t, temp, "percentage"); got != 150 {
		t.Errorf("TempRateResponse.percentage = %d, want 150", got)
	}

	historyBefore := readBack(t, bridge, handler.NewHistoryLogStatusHandler(bridge), ps)
	countBefore := cargoInt(t, historyBefore, "numEntries")

	mustDo(t, mux, http.MethodPost, "/api/state/suspend", `{"reason":"occlusion"}`)

	// After it: suspended, no temp rate, no bolus in progress.
	basal = readBack(t, bridge, handler.NewCurrentBasalStatusHandler(bridge), ps)
	assertCargo(t, basal, "basalModifiedBitmask", 0x01)
	assertCargo(t, basal, "currentBasalRate", 0)
	assertCargo(t, basal, "profileBasalRate", 1000)

	bolus = readBack(t, bridge, handler.NewCurrentBolusStatusHandler(bridge), ps)
	assertCargo(t, bolus, "statusId", 0)
	assertCargo(t, bolus, "bolusId", 0)

	// The bolus the suspend cut short is reported as the last bolus, with the
	// id the driver knows and the volume that actually went in.
	last := readBack(t, bridge, handler.NewLastBolusStatusHandler(bridge, "LastBolusStatusV2Request"), ps)
	assertCargo(t, last, "bolusId", bolusID)
	assertCargo(t, last, "bolusStatusId", state.BolusEndReasonStopped)
	assertCargo(t, last, "requestedVolume", 5000)
	if delivered := cargoInt(t, last, "deliveredVolume"); delivered < 990 || delivered > 1010 {
		t.Errorf("LastBolusStatusV2Response.deliveredVolume = %d mU, want ~1000", delivered)
	}
	// The end timestamp is the pump's, in pump-epoch seconds, taken from the
	// controllable clock.
	assertCargo(t, last, "timestamp", int64(ps.PumpTimeFor(testInstant.Add(10e9))))

	historyAfter := readBack(t, bridge, handler.NewHistoryLogStatusHandler(bridge), ps)
	if got := cargoInt(t, historyAfter, "numEntries"); got <= countBefore {
		t.Errorf("history entries went %d -> %d; the suspend recorded nothing", countBefore, got)
	}
	if got := cargoInt(t, historyAfter, "lastSequenceNum"); got != cargoInt(t, historyAfter, "numEntries") {
		t.Errorf("HistoryLogStatusResponse is internally inconsistent: %v", historyAfter.Cargo)
	}
}

// TestResumeActionIsVisibleToTheDriver checks the other half: after a resume,
// basal reads as ordinary profile basal again.
func TestResumeActionIsVisibleToTheDriver(t *testing.T) {
	bridge := testBridge(t)
	_, ps, _, mux := testHarness(t)

	mustDo(t, mux, http.MethodPut, "/api/state", `{"basal_rate":0.85}`)
	mustDo(t, mux, http.MethodPost, "/api/state/suspend", `{"reason":"user"}`)
	mustDo(t, mux, http.MethodPost, "/api/state/resume", "")

	basal := readBack(t, bridge, handler.NewCurrentBasalStatusHandler(bridge), ps)
	assertCargo(t, basal, "basalModifiedBitmask", 0)
	assertCargo(t, basal, "currentBasalRate", 850)
}

// TestBolusCompletionIsVisibleToTheDriver walks a pump-initiated bolus to
// completion on the controllable clock and reads the result back.
func TestBolusCompletionIsVisibleToTheDriver(t *testing.T) {
	bridge := testBridge(t)
	_, ps, _, mux := testHarness(t)

	mustDo(t, mux, http.MethodPut, "/api/clock", `{"mode":"manual","frozen":true,"now":"2024-03-05T12:00:00Z"}`)
	body := mustDo(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":3,"rate":0.1,"source":"pump"}`)
	bolusID := int64(body["bolus_id"].(float64))

	inProgress := readBack(t, bridge, handler.NewCurrentBolusStatusHandler(bridge), ps)
	assertCargo(t, inProgress, "requestedVolume", 3000)
	// A bolus the pump started itself must not claim the app commanded it.
	assertCargo(t, inProgress, "bolusSourceId", state.BolusSourceQuickBolus)

	mustDo(t, mux, http.MethodPost, "/api/clock/advance", `{"seconds":30}`)
	if ps.IsBolusActive() {
		t.Fatal("the bolus is still running after enough pump time to finish it")
	}

	last := readBack(t, bridge, handler.NewLastBolusStatusHandler(bridge, "LastBolusStatusV2Request"), ps)
	assertCargo(t, last, "bolusId", bolusID)
	assertCargo(t, last, "deliveredVolume", 3000)
	assertCargo(t, last, "bolusStatusId", state.BolusEndReasonCompleted)
	assertCargo(t, last, "bolusSourceId", state.BolusSourceQuickBolus)
	assertCargo(t, last, "timestamp", int64(ps.PumpTimeFor(testInstant.Add(30e9))))
}

// TestPumpClockSkewShiftsEveryEmittedTimestamp checks that the offset is a
// pump-clock lie rather than a change to the emulator's own time: the dose
// timestamps move, the delivery arithmetic does not.
func TestPumpClockSkewShiftsEveryEmittedTimestamp(t *testing.T) {
	bridge := testBridge(t)
	_, ps, _, mux := testHarness(t)

	mustDo(t, mux, http.MethodPut, "/api/clock",
		`{"mode":"manual","frozen":true,"now":"2024-03-05T12:00:00Z","pump_offset_seconds":8}`)
	mustDo(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":3,"rate":0.1}`)

	bolus := readBack(t, bridge, handler.NewCurrentBolusStatusHandler(bridge), ps)
	assertCargo(t, bolus, "timestamp", int64(state.PumpTimeSecondsIn(testInstant, ps.GetPumpTimeZone()))+8)

	timeMsg := readBack(t, bridge, handler.NewTimeSinceResetHandler(bridge), ps)
	assertCargo(t, timeMsg, "currentTime", int64(state.PumpTimeSecondsIn(testInstant, ps.GetPumpTimeZone()))+8)

	mustDo(t, mux, http.MethodPost, "/api/clock/advance", `{"seconds":30}`)
	last := readBack(t, bridge, handler.NewLastBolusStatusHandler(bridge, "LastBolusStatusV2Request"), ps)
	assertCargo(t, last, "timestamp", int64(state.PumpTimeSecondsIn(testInstant.Add(30e9), ps.GetPumpTimeZone()))+8)
	assertCargo(t, last, "deliveredVolume", 3000)
}
