package harness

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// recordTime reads a history record's true wall-clock instant.
func recordTime(t *testing.T, entry map[string]interface{}) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339Nano, entry["time"].(string))
	if err != nil {
		t.Fatalf("record %v has an unparseable time: %v", entry["sequence"], err)
	}
	return parsed.UTC()
}

// assertTempRateCompleted checks what a timeline needs from a
// TempRateCompleted: the temp rate it closes, the instant it closed, and how
// much of the programmed duration was still to run. tempRateId and timeLeft are
// the only fields the record format carries; everything descriptive is in the
// JSON-only extra.
func assertTempRateCompleted(t *testing.T, entry map[string]interface{}, tempRateID int, when time.Time, timeLeft int) {
	t.Helper()

	data := fieldsOf(entry, "data")
	if got, want := len(data), 2; got != want {
		t.Errorf("TempRateCompleted has %d wire fields, want exactly %d; has %v", got, want, data)
	}
	if got := data["tempRateId"]; got != float64(tempRateID) {
		t.Errorf("TempRateCompleted tempRateId = %v, want the %d it closed", got, tempRateID)
	}
	if got := data["timeLeft"]; got != float64(timeLeft) {
		t.Errorf("TempRateCompleted timeLeft = %v, want %d seconds", got, timeLeft)
	}
	if got := recordTime(t, entry); !got.Equal(when) {
		t.Errorf("TempRateCompleted stamped at %v, want the true end instant %v", got, when)
	}
	extra := fieldsOf(entry, "extra")
	for _, name := range []string{"tempRate", "normalRate", "percent", "durationSeconds", "deliveredUnits", "endedEarly"} {
		if _, ok := extra[name]; !ok {
			t.Errorf("TempRateCompleted extra is missing %q; has %v", name, extra)
		}
	}
}

func advance(t *testing.T, mux http.Handler, seconds int) {
	t.Helper()
	mustDo(t, mux, http.MethodPost, "/api/clock/advance", fmt.Sprintf(`{"seconds":%d}`, seconds))
}

// TestHarnessTempRateEndingsAreRecordedOnce covers the two harness paths that
// end a temp rate: a replacement started while one is running (the old one ends
// at the replacement instant) and the explicit stop action. Each writes exactly
// one TempRateCompleted, and neither writes a second.
func TestHarnessTempRateEndingsAreRecordedOnce(t *testing.T) {
	_, _, _, mux := testHarness(t)
	putManualClock(t, mux)

	first := mustDo(t, mux, http.MethodPost, "/api/state/tempbasal/start",
		`{"percent":150,"duration_minutes":30}`)
	firstID := int(first["temp_rate_id"].(float64))

	advance(t, mux, 300)
	replacedAt := testInstant.Add(5 * time.Minute)

	second := mustDo(t, mux, http.MethodPost, "/api/state/tempbasal/start",
		`{"percent":50,"duration_minutes":60}`)
	secondID := int(second["temp_rate_id"].(float64))
	if secondID == firstID {
		t.Fatalf("the replacement temp rate reused id %d", firstID)
	}

	entries := historyEntries(t, mux)
	completed := recordsOfType(entries, "TempRateCompleted")
	if len(completed) != 1 {
		t.Fatalf("replacing a temp rate wrote %d TempRateCompleted records, want exactly 1", len(completed))
	}
	assertTempRateCompleted(t, completed[0], firstID, replacedAt, 25*60)

	// The replaced rate closes before the replacement opens, so a timeline
	// reading the log in order never sees two temp rates running at once.
	activated := recordsOfType(entries, "TempRateActivated")
	if len(activated) != 2 {
		t.Fatalf("got %d TempRateActivated records, want 2", len(activated))
	}
	if completed[0]["sequence"].(float64) > activated[1]["sequence"].(float64) {
		t.Error("the replaced temp rate's completion was logged after the replacement's activation")
	}

	advance(t, mux, 600)
	stoppedAt := testInstant.Add(15 * time.Minute)
	mustDo(t, mux, http.MethodPost, "/api/state/tempbasal/stop", "")

	completed = recordsOfType(historyEntries(t, mux), "TempRateCompleted")
	if len(completed) != 2 {
		t.Fatalf("got %d TempRateCompleted records after the stop, want 2", len(completed))
	}
	assertTempRateCompleted(t, completed[1], secondID, stoppedAt, 50*60)

	// Nothing is running now, so stopping again is a conflict and writes
	// nothing.
	code, _ := do(t, mux, http.MethodPost, "/api/state/tempbasal/stop", "")
	if code != http.StatusConflict {
		t.Errorf("stopping with no temp rate running = %d, want %d", code, http.StatusConflict)
	}
	if got := len(recordsOfType(historyEntries(t, mux), "TempRateCompleted")); got != 2 {
		t.Errorf("a refused stop brought the total to %d TempRateCompleted records, want 2", got)
	}
}

// TestHarnessSuspendEndsEverythingOnce: a pump-initiated stop ends an
// in-progress bolus and a running temp rate, each with exactly one record,
// written before the PumpingSuspended record that caused them. A second suspend
// is refused and writes nothing.
func TestHarnessSuspendEndsEverythingOnce(t *testing.T) {
	_, ps, _, mux := testHarness(t)
	putManualClock(t, mux)

	// Slow enough that the bolus is still running when the suspend lands.
	mustDo(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":5.0,"rate":0.01,"source":"remote"}`)
	temp := mustDo(t, mux, http.MethodPost, "/api/state/tempbasal/start", `{"percent":150,"duration_minutes":30}`)
	tempID := int(temp["temp_rate_id"].(float64))

	advance(t, mux, 60)
	suspendedAt := testInstant.Add(time.Minute)
	mustDo(t, mux, http.MethodPost, "/api/state/suspend", `{"reason":"occlusion"}`)

	entries := historyEntries(t, mux)

	bolusEnds := recordsOfType(entries, "BolusCompleted")
	if len(bolusEnds) != 1 {
		t.Fatalf("the suspend wrote %d BolusCompleted records, want exactly 1", len(bolusEnds))
	}
	if got := fieldsOf(bolusEnds[0], "data")["completionStatusId"]; got != float64(0) {
		t.Errorf("completionStatusId = %v, want 0 (STOPPED_USER_TERMINATED)", got)
	}

	tempEnds := recordsOfType(entries, "TempRateCompleted")
	if len(tempEnds) != 1 {
		t.Fatalf("the suspend wrote %d TempRateCompleted records, want exactly 1", len(tempEnds))
	}
	assertTempRateCompleted(t, tempEnds[0], tempID, suspendedAt, 29*60)

	suspends := recordsOfType(entries, "PumpingSuspended")
	if len(suspends) != 1 {
		t.Fatalf("got %d PumpingSuspended records, want exactly 1", len(suspends))
	}
	if suspends[0]["sequence"].(float64) < tempEnds[0]["sequence"].(float64) ||
		suspends[0]["sequence"].(float64) < bolusEnds[0]["sequence"].(float64) {
		t.Error("PumpingSuspended was logged before the endings it caused")
	}
	if ps.GetTempRate().Active || ps.IsBolusActive() {
		t.Error("delivery is suspended but the bolus or temp rate is still running")
	}

	code, _ := do(t, mux, http.MethodPost, "/api/state/suspend", `{"reason":"user"}`)
	if code != http.StatusConflict {
		t.Errorf("suspending an already-suspended pump = %d, want %d", code, http.StatusConflict)
	}

	mustDo(t, mux, http.MethodPost, "/api/state/resume", "")
	code, _ = do(t, mux, http.MethodPost, "/api/state/resume", "")
	if code != http.StatusConflict {
		t.Errorf("resuming a running pump = %d, want %d", code, http.StatusConflict)
	}

	entries = historyEntries(t, mux)
	for _, typeName := range []string{"BolusCompleted", "TempRateCompleted", "PumpingSuspended", "PumpingResumed"} {
		if got := len(recordsOfType(entries, typeName)); got != 1 {
			t.Errorf("%d %s records after the refused repeats, want exactly 1", got, typeName)
		}
	}
}

// TestHarnessTempRateSurvivingToExpiryIsRecordedOnce keeps the natural-expiry
// path honest alongside the ones that cut a temp rate short: the simulator
// records it once, at the programmed end rather than at the tick that noticed,
// and the stop action afterwards has nothing left to end.
func TestHarnessTempRateSurvivingToExpiryIsRecordedOnce(t *testing.T) {
	_, _, _, mux := testHarness(t)
	putManualClock(t, mux)

	temp := mustDo(t, mux, http.MethodPost, "/api/state/tempbasal/start",
		`{"percent":150,"duration_minutes":30}`)
	tempID := int(temp["temp_rate_id"].(float64))

	// One coarse step, well past the programmed end.
	advance(t, mux, 45*60)

	completed := recordsOfType(historyEntries(t, mux), "TempRateCompleted")
	if len(completed) != 1 {
		t.Fatalf("expiry wrote %d TempRateCompleted records, want exactly 1", len(completed))
	}
	assertTempRateCompleted(t, completed[0], tempID, testInstant.Add(30*time.Minute), 0)
	if got := fieldsOf(completed[0], "extra")["endedEarly"]; got != false {
		t.Errorf("extra endedEarly = %v, want false for a temp rate that ran its course", got)
	}

	code, _ := do(t, mux, http.MethodPost, "/api/state/tempbasal/stop", "")
	if code != http.StatusConflict {
		t.Errorf("stopping an expired temp rate = %d, want %d", code, http.StatusConflict)
	}
	if got := len(recordsOfType(historyEntries(t, mux), "TempRateCompleted")); got != 1 {
		t.Errorf("%d TempRateCompleted records after the refused stop, want 1", got)
	}
}

// TestHarnessBolusEndingsAreRecordedOnce: aborting and completing each write
// one BolusCompleted with the right end reason, and a repeat is refused.
func TestHarnessBolusEndingsAreRecordedOnce(t *testing.T) {
	_, _, _, mux := testHarness(t)
	putManualClock(t, mux)

	mustDo(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":4.0,"rate":0.01,"source":"remote"}`)
	advance(t, mux, 30)
	mustDo(t, mux, http.MethodPost, "/api/state/bolus/abort", `{"delivered_units":1.25}`)

	code, _ := do(t, mux, http.MethodPost, "/api/state/bolus/abort", "")
	if code != http.StatusConflict {
		t.Errorf("aborting with no bolus running = %d, want %d", code, http.StatusConflict)
	}

	completed := recordsOfType(historyEntries(t, mux), "BolusCompleted")
	if len(completed) != 1 {
		t.Fatalf("aborting wrote %d BolusCompleted records, want exactly 1", len(completed))
	}
	data := fieldsOf(completed[0], "data")
	if got := data["completionStatusId"]; got != float64(0) {
		t.Errorf("an aborted bolus completionStatusId = %v, want 0 (STOPPED_USER_TERMINATED)", got)
	}
	if got := data["insulinDelivered"]; got != 1.25 {
		t.Errorf("insulinDelivered = %v, want the declared 1.25", got)
	}

	// A second bolus, run to completion, is the other end reason -- and again
	// exactly one record.
	mustDo(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":2.0,"rate":0.01,"source":"remote"}`)
	mustDo(t, mux, http.MethodPost, "/api/state/bolus/complete", "")

	completed = recordsOfType(historyEntries(t, mux), "BolusCompleted")
	if len(completed) != 2 {
		t.Fatalf("got %d BolusCompleted records, want 2", len(completed))
	}
	data = fieldsOf(completed[1], "data")
	if got := data["completionStatusId"]; got != float64(3) {
		t.Errorf("a completed bolus completionStatusId = %v, want 3 (COMPLETE)", got)
	}
	if got := data["insulinDelivered"]; got != 2.0 {
		t.Errorf("insulinDelivered = %v, want the full 2.0 units", got)
	}
}
