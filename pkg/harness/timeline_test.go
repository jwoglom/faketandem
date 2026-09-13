package harness

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/state"
)

// tandemEpochUnix is the Tandem pump epoch in Unix seconds, repeated here so
// the test decodes wire values the way a driver does rather than by calling the
// same helper the emulator encoded with.
const tandemEpochUnix = 1199145600

// decodeWire is TandemKit's Dates.fromJan12008ToUnixEpochSeconds with the
// pump's reported zone offset standing in for the phone's.
func decodeWire(pumpSeconds, zoneOffsetSeconds float64) time.Time {
	return time.Unix(int64(pumpSeconds)+tandemEpochUnix-int64(zoneOffsetSeconds), 0).UTC()
}

// putManualClock puts the harness's own manual clock in charge at testInstant,
// which is what /api/clock/advance needs -- testHarness only installs a frozen
// clock on the pump state itself.
func putManualClock(t *testing.T, mux http.Handler) {
	t.Helper()
	mustDo(t, mux, http.MethodPut, "/api/clock",
		fmt.Sprintf(`{"mode":"manual","now":%q,"frozen":true}`, testInstant.Format(time.RFC3339)))
}

func clockOffsets(t *testing.T, mux http.Handler) (zoneOffset, skew float64) {
	t.Helper()
	clock := mustDo(t, mux, http.MethodGet, "/api/clock", "")
	return clock["pump_timezone_offset_seconds"].(float64), clock["pump_offset_seconds"].(float64)
}

// historyEntries reads every record from GET /api/history.
func historyEntries(t *testing.T, mux http.Handler) []map[string]interface{} {
	t.Helper()
	body := mustDo(t, mux, http.MethodGet, "/api/history?limit=1000", "")
	raw := body["entries"].([]interface{})
	out := make([]map[string]interface{}, 0, len(raw))
	for _, e := range raw {
		out = append(out, e.(map[string]interface{}))
	}
	return out
}

// recordsOfType returns every history record with the given type name.
func recordsOfType(entries []map[string]interface{}, typeName string) []map[string]interface{} {
	var out []map[string]interface{}
	for _, e := range entries {
		if e["type"] == typeName {
			out = append(out, e)
		}
	}
	return out
}

func fieldsOf(entry map[string]interface{}, key string) map[string]interface{} {
	if raw, ok := entry[key].(map[string]interface{}); ok {
		return raw
	}
	return map[string]interface{}{}
}

func requireFields(t *testing.T, entry map[string]interface{}, names ...string) {
	t.Helper()
	data := fieldsOf(entry, "data")
	for _, name := range names {
		if _, ok := data[name]; !ok {
			t.Errorf("%s record is missing wire field %q; has %v", entry["type"], name, data)
		}
	}
}

// TestSnapshotTimesFollowOneRule is the pin on gap 2. One rule, every path:
//
//	wall-clock fields are true instants on the pump's Clock;
//	*_pump_seconds / pump_seconds fields are what went on the wire.
//
// So decoding any wire field with the offsets /api/clock reports must land on
// the matching wall-clock field, whatever produced the record. Before this,
// AddHistoryLogEntryAt rebuilt a record's wall-clock time *from* the already
// skewed wire value while the protocol path stamped the true instant, so the
// two paths disagreed by the skew.
func TestSnapshotTimesFollowOneRule(t *testing.T) {
	_, _, _, mux := testHarness(t)

	// A skew big enough that a violation cannot hide in rounding, on top of
	// whatever local-time offset the pump's zone contributes.
	mustDo(t, mux, http.MethodPut, "/api/clock",
		fmt.Sprintf(`{"mode":"manual","now":%q,"frozen":true,"pump_offset_seconds":3600}`,
			testInstant.Format(time.RFC3339)))

	// Three paths into the log: a harness action, a backdated staged record,
	// and the simulator finishing a bolus on a tick.
	mustDo(t, mux, http.MethodPost, "/api/state/history/append",
		`{"type":"BolusCompleted","seconds_ago":600,"data":{"bolusId":7}}`)
	mustDo(t, mux, http.MethodPost, "/api/state/tempbasal/start", `{"percent":150,"duration_minutes":30}`)
	mustDo(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":1.0,"rate":1.0}`)
	// 1 U at 1 U/s: the simulator finishes it inside this step.
	mustDo(t, mux, http.MethodPost, "/api/clock/advance", `{"seconds":5}`)

	zoneOffset, skew := clockOffsets(t, mux)

	for _, entry := range historyEntries(t, mux) {
		wall, err := time.Parse(time.RFC3339Nano, entry["time"].(string))
		if err != nil {
			t.Fatalf("record %v has an unparseable time: %v", entry["sequence"], err)
		}
		decoded := decodeWire(entry["pump_seconds"].(float64), zoneOffset+skew)
		if !decoded.Equal(wall.UTC()) {
			t.Errorf("%s record %v: wire %v decodes to %v but its wall-clock time says %v",
				entry["type"], entry["sequence"], entry["pump_seconds"], decoded, wall.UTC())
		}
		if entry["pump_time"] != entry["pump_seconds"] {
			t.Errorf("record %v: pump_time and pump_seconds disagree", entry["sequence"])
		}
	}

	// The same rule for the live snapshot's paired fields.
	snapshot := mustDo(t, mux, http.MethodGet, "/api/state", "")
	pairs := []struct {
		section       map[string]interface{}
		wallKey, wire string
	}{
		{snapshot["last_bolus"].(map[string]interface{}), "end_time", "end_pump_seconds"},
		{snapshot["basal"].(map[string]interface{}), "temp_start", "temp_start_pump_seconds"},
	}
	for _, pair := range pairs {
		raw, ok := pair.section[pair.wallKey].(string)
		if !ok || raw == "" {
			t.Fatalf("snapshot is missing %s, so the rule cannot be checked", pair.wallKey)
		}
		wall, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			t.Fatalf("%s is unparseable: %v", pair.wallKey, err)
		}
		decoded := decodeWire(pair.section[pair.wire].(float64), zoneOffset+skew)
		if !decoded.Equal(wall.UTC()) {
			t.Errorf("%s/%s disagree: wire decodes to %v, wall-clock says %v",
				pair.wallKey, pair.wire, decoded, wall.UTC())
		}
	}

	// And the wall-clock fields really are the emulator's own clock, not a
	// skewed one: the staged record sits exactly ten minutes before "now".
	staged := recordsOfType(historyEntries(t, mux), "BolusCompleted")[0]
	stagedWall, _ := time.Parse(time.RFC3339Nano, staged["time"].(string))
	if want := testInstant.Add(-10 * time.Minute); !stagedWall.UTC().Equal(want) {
		t.Errorf("backdated record's wall-clock time = %v, want %v (unskewed)", stagedWall.UTC(), want)
	}
}

// TestHistoryRecordFieldsAreTheSameWhicheverPathWroteThem is the pin on gap 3.
// The harness path and the protocol path must name a record's fields
// identically, and those names must be the ones pumpX2 and TandemKit use. The
// protocol side of the same pin is TestProtocolPathWritesPumpX2FieldNames in
// pkg/handler.
func TestHistoryRecordFieldsAreTheSameWhicheverPathWroteThem(t *testing.T) {
	_, _, _, mux := testHarness(t)

	mustDo(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":2.0,"rate":2.0,"source":"remote"}`)
	mustDo(t, mux, http.MethodPost, "/api/state/bolus/abort", "")
	mustDo(t, mux, http.MethodPost, "/api/state/tempbasal/start", `{"percent":150,"duration_minutes":30}`)
	mustDo(t, mux, http.MethodPost, "/api/state/tempbasal/stop", "")
	mustDo(t, mux, http.MethodPost, "/api/state/suspend", `{"reason":"occlusion"}`)
	mustDo(t, mux, http.MethodPost, "/api/state/resume", "")

	entries := historyEntries(t, mux)

	t.Run("BolusActivated", func(t *testing.T) { assertBolusActivatedFields(t, entries) })
	t.Run("BolusCompleted", func(t *testing.T) { assertBolusCompletedFields(t, entries) })
	t.Run("TempRate", func(t *testing.T) { assertTempRateFields(t, entries) })
	t.Run("PumpingSuspended", func(t *testing.T) { assertSuspendFields(t, entries) })
}

// assertBolusActivatedFields: the wire carries bolusId/selectedIob/iob/bolusSize,
// and the bolus source (which is not a field of this record -- it rides on
// BolusDeliveryHistoryLog) is context only.
func assertBolusActivatedFields(t *testing.T, entries []map[string]interface{}) {
	t.Helper()

	activated := recordsOfType(entries, "BolusActivated")
	if len(activated) != 1 {
		t.Fatalf("expected one BolusActivated, got %d", len(activated))
	}
	requireFields(t, activated[0], "bolusId", "selectedIob", "iob", "bolusSize")
	if _, present := fieldsOf(activated[0], "data")["units"]; present {
		t.Error("BolusActivated still carries the harness-only `units` field")
	}
	if got := fieldsOf(activated[0], "extra")["bolusSourceId"]; got != float64(state.BolusSourceBluetoothRemote) {
		t.Errorf("BolusActivated extra bolusSourceId = %v, want %d", got, state.BolusSourceBluetoothRemote)
	}
}

// assertBolusCompletedFields: completionStatusId IS the end reason.
func assertBolusCompletedFields(t *testing.T, entries []map[string]interface{}) {
	t.Helper()

	completed := recordsOfType(entries, "BolusCompleted")
	if len(completed) != 1 {
		t.Fatalf("expected one BolusCompleted, got %d", len(completed))
	}
	requireFields(t, completed[0], "completionStatusId", "bolusId", "iob", "insulinDelivered", "insulinRequested")
	if got := fieldsOf(completed[0], "data")["completionStatusId"]; got != float64(state.BolusEndReasonStopped) {
		t.Errorf("aborted bolus completionStatusId = %v, want %d", got, state.BolusEndReasonStopped)
	}
}

// assertTempRateFields: TempRateActivated carries percent/durationMilliseconds/
// tempRateId, and TempRateCompleted carries tempRateId and timeLeft -- that is
// the entire record format -- with the rates, the start and the delivered
// volume reported alongside it rather than invented onto the wire.
func assertTempRateFields(t *testing.T, entries []map[string]interface{}) {
	t.Helper()

	tempOn := recordsOfType(entries, "TempRateActivated")[0]
	requireFields(t, tempOn, "percent", "durationMilliseconds", "tempRateId")
	if _, present := fieldsOf(tempOn, "data")["minutes"]; present {
		t.Error("TempRateActivated still carries the harness-only `minutes` field")
	}
	if got := fieldsOf(tempOn, "data")["durationMilliseconds"]; got != float64(30*60*1000) {
		t.Errorf("durationMilliseconds = %v, want %d", got, 30*60*1000)
	}
	for _, name := range []string{"tempRate", "normalRate"} {
		if _, ok := fieldsOf(tempOn, "extra")[name]; !ok {
			t.Errorf("TempRateActivated extra is missing %q", name)
		}
	}

	tempOff := recordsOfType(entries, "TempRateCompleted")[0]
	requireFields(t, tempOff, "tempRateId", "timeLeft")
	if got, want := len(fieldsOf(tempOff, "data")), 2; got != want {
		t.Errorf("TempRateCompleted has %d wire fields, want exactly %d (the record format has no others)", got, want)
	}
	if got := fieldsOf(tempOff, "data")["tempRateId"]; got != fieldsOf(tempOn, "data")["tempRateId"] {
		t.Errorf("TempRateCompleted tempRateId %v does not name the temp rate it closed (%v)",
			got, fieldsOf(tempOn, "data")["tempRateId"])
	}
	extra := fieldsOf(tempOff, "extra")
	for _, name := range []string{"tempRate", "normalRate", "startPumpSeconds", "deliveredUnits"} {
		if _, ok := extra[name]; !ok {
			t.Errorf("TempRateCompleted extra is missing %q", name)
		}
	}
	// Stopped straight away, so nearly the whole 30 minutes was left.
	if left := fieldsOf(tempOff, "data")["timeLeft"].(float64); left < 1700 || left > 1800 {
		t.Errorf("timeLeft = %v s, want close to the unrun 1800 s", left)
	}
}

// assertSuspendFields: the wire carries reasonId, and the reason string is
// context. An occlusion is a malfunction-class stop.
func assertSuspendFields(t *testing.T, entries []map[string]interface{}) {
	t.Helper()

	suspended := recordsOfType(entries, "PumpingSuspended")[0]
	requireFields(t, suspended, "preSuspendState", "insulinAmount", "reasonId", "rpaTimeout")
	if got := fieldsOf(suspended, "data")["reasonId"]; got != float64(state.SuspendReasonIDMalfunction) {
		t.Errorf("occlusion suspend reasonId = %v, want %d", got, state.SuspendReasonIDMalfunction)
	}
	if _, present := fieldsOf(suspended, "data")["reason"]; present {
		t.Error("PumpingSuspended still carries the harness-only `reason` string as a wire field")
	}
	if got := fieldsOf(suspended, "extra")["reason"]; got != SuspendReasonOcclusion {
		t.Errorf("PumpingSuspended extra reason = %v, want %q", got, SuspendReasonOcclusion)
	}
}

// TestSimulatorCompletionCarriesAnEndReason pins the simulator's own bolus
// completion, which used to omit the end reason entirely -- making a bolus that
// ran to completion indistinguishable, in the log, from one that was canceled.
func TestSimulatorCompletionCarriesAnEndReason(t *testing.T) {
	_, _, _, mux := testHarness(t)
	putManualClock(t, mux)

	mustDo(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":1.0,"rate":1.0}`)
	mustDo(t, mux, http.MethodPost, "/api/clock/advance", `{"seconds":5}`)

	completed := recordsOfType(historyEntries(t, mux), "BolusCompleted")
	if len(completed) != 1 {
		t.Fatalf("expected the simulator to have completed the bolus, got %d records", len(completed))
	}
	requireFields(t, completed[0], "completionStatusId", "bolusId", "insulinDelivered", "insulinRequested")
	if got := fieldsOf(completed[0], "data")["completionStatusId"]; got != float64(state.BolusEndReasonCompleted) {
		t.Errorf("completionStatusId = %v, want %d (completed)", got, state.BolusEndReasonCompleted)
	}
}

// TestHistoryEndpointPagesTheWholeLog is the pin on gap 4: /api/history is a
// feed, not the 50-record tail, and it reports what the tail could not.
func TestHistoryEndpointPagesTheWholeLog(t *testing.T) {
	_, ps, _, mux := testHarness(t)

	for i := 0; i < 120; i++ {
		ps.RecordPumpingResumed()
	}

	first := mustDo(t, mux, http.MethodGet, "/api/history?limit=50", "")
	if got := len(first["entries"].([]interface{})); got != 50 {
		t.Fatalf("first page has %d entries, want 50", got)
	}
	if first["truncated"] != true {
		t.Error("a cut-short page should report truncated")
	}
	if got, want := int(first["count"].(float64)), 120; got != want {
		t.Errorf("count = %d, want the whole log's %d", got, want)
	}

	// Paging with next_since walks the log exactly once, in order.
	seen := 0
	since := first["next_since"].(float64)
	seen += 50
	for {
		page := mustDo(t, mux, http.MethodGet,
			fmt.Sprintf("/api/history?since=%d&limit=50", int(since)), "")
		entries := page["entries"].([]interface{})
		if len(entries) == 0 {
			break
		}
		seen += len(entries)
		since = page["next_since"].(float64)
	}
	if seen != 120 {
		t.Errorf("paging saw %d records, want all 120", seen)
	}

	// The tail in /api/state is unchanged and still capped.
	snapshot := mustDo(t, mux, http.MethodGet, "/api/state", "")
	tail := snapshot["history"].(map[string]interface{})["entries"].([]interface{})
	if len(tail) != historySnapshotLimit {
		t.Errorf("state tail has %d entries, want the unchanged %d-record cap", len(tail), historySnapshotLimit)
	}

	// Each entry carries the type id, both timestamps and the source nibble.
	entry := first["entries"].([]interface{})[0].(map[string]interface{})
	for _, key := range []string{"sequence", "type_id", "type", "time", "pump_seconds", "pump_time", "source_nibble"} {
		if _, ok := entry[key]; !ok {
			t.Errorf("history entry is missing %q: %v", key, entry)
		}
	}
}

// TestAlarmsAreReportedWithBothEnds is the pin on gap 5: an alarm that came and
// went used to leave nothing a timeline could place, because `alerts` reports
// only what is standing and the clearing record said just "3 alarms went away".
func TestAlarmsAreReportedWithBothEnds(t *testing.T) {
	_, _, _, mux := testHarness(t)
	putManualClock(t, mux)

	mustDo(t, mux, http.MethodPost, "/api/state/suspend", `{"reason":"occlusion"}`)

	snapshot := mustDo(t, mux, http.MethodGet, "/api/state", "")
	active := snapshot["alerts"].([]interface{})
	if len(active) != 1 {
		t.Fatalf("expected one standing alarm, got %d", len(active))
	}
	raised := active[0].(map[string]interface{})
	if raised["type_name"] != "Occlusion" {
		t.Errorf("standing alarm type_name = %v, want Occlusion", raised["type_name"])
	}

	// The AlarmActivated record names the alarm and reports the alert type.
	activated := recordsOfType(historyEntries(t, mux), "AlarmActivated")
	if len(activated) != 1 {
		t.Fatalf("expected one AlarmActivated, got %d", len(activated))
	}
	requireFields(t, activated[0], "alarmId")
	if got, want := fieldsOf(activated[0], "data")["alarmId"], raised["id"]; got != want {
		t.Errorf("AlarmActivated alarmId = %v, want the alarm's own id %v", got, want)
	}
	if got := fieldsOf(activated[0], "extra")["alertTypeId"]; got != float64(state.AlertOcclusion) {
		t.Errorf("AlarmActivated extra alertTypeId = %v, want %d", got, state.AlertOcclusion)
	}

	mustDo(t, mux, http.MethodPost, "/api/clock/advance", `{"seconds":90}`)
	mustDo(t, mux, http.MethodPost, "/api/state/resume", "")

	snapshot = mustDo(t, mux, http.MethodGet, "/api/state", "")
	if got := len(snapshot["alerts"].([]interface{})); got != 0 {
		t.Errorf("%d alarms still standing after a resume", got)
	}
	cleared := snapshot["alarms_history"].([]interface{})
	if len(cleared) != 1 {
		t.Fatalf("alarms_history has %d entries, want the one alarm that came and went", len(cleared))
	}
	record := cleared[0].(map[string]interface{})
	if record["type_name"] != "Occlusion" || record["id"] != raised["id"] {
		t.Errorf("cleared alarm = %v, want the occlusion that was raised", record)
	}

	zoneOffset, skew := clockOffsets(t, mux)
	raisedAt := decodeWire(record["pump_seconds"].(float64), zoneOffset+skew)
	clearedAt := decodeWire(record["cleared_pump_seconds"].(float64), zoneOffset+skew)
	if got := clearedAt.Sub(raisedAt); got != 90*time.Second {
		t.Errorf("alarm interval = %v, want the 90 s it stood for", got)
	}

	// One AlarmCleared record per alarm, naming the alarm it closes.
	clearedRecords := recordsOfType(historyEntries(t, mux), "AlarmCleared")
	if len(clearedRecords) != 1 {
		t.Fatalf("expected one AlarmCleared record, got %d", len(clearedRecords))
	}
	if got, want := fieldsOf(clearedRecords[0], "data")["alarmId"], raised["id"]; got != want {
		t.Errorf("AlarmCleared alarmId = %v, want %v", got, want)
	}
}

// TestClockAcceptsAPumpTimeZone pins the new /api/clock field, including that
// moving the pump's zone moves every wire timestamp and nothing else.
func TestClockAcceptsAPumpTimeZone(t *testing.T) {
	_, ps, _, mux := testHarness(t)

	body := mustDo(t, mux, http.MethodPut, "/api/clock",
		fmt.Sprintf(`{"mode":"manual","now":%q,"frozen":true,"pump_timezone":"America/New_York"}`,
			testInstant.Format(time.RFC3339)))

	if body["pump_timezone"] != "America/New_York" {
		t.Fatalf("pump_timezone = %v", body["pump_timezone"])
	}
	// 2024-03-05 is EST, five hours behind UTC.
	if got, want := body["pump_timezone_offset_seconds"].(float64), float64(-5*3600); got != want {
		t.Errorf("pump_timezone_offset_seconds = %v, want %v", got, want)
	}
	wire := body["pump_time_seconds"].(float64)
	if got := decodeWire(wire, body["pump_timezone_offset_seconds"].(float64)); !got.Equal(testInstant) {
		t.Errorf("the reported wire time decodes to %v, want %v", got, testInstant)
	}
	// The pump's own clock did not move.
	if got := ps.Now(); !got.Equal(testInstant) {
		t.Errorf("Now() = %v, want %v", got, testInstant)
	}

	// Moving the pump to UTC removes the offset entirely.
	body = mustDo(t, mux, http.MethodPut, "/api/clock", `{"mode":"manual","pump_timezone":"UTC"}`)
	if got := body["pump_timezone_offset_seconds"].(float64); got != 0 {
		t.Errorf("UTC offset = %v, want 0", got)
	}
	if got := body["pump_time_seconds"].(float64); got != wire+5*3600 {
		t.Errorf("UTC wire time = %v, want %v", got, wire+5*3600)
	}

	if code, _ := do(t, mux, http.MethodPut, "/api/clock", `{"mode":"manual","pump_timezone":"Mars/Olympus"}`); code != http.StatusBadRequest {
		t.Errorf("an unknown zone = %d, want 400", code)
	}
}
