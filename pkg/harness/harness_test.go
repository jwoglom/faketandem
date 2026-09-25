package harness

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/faults"
	"github.com/jwoglom/faketandem/pkg/handler"
	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/reqlog"
	"github.com/jwoglom/faketandem/pkg/state"
)

var testInstant = time.Date(2024, time.March, 5, 12, 0, 0, 0, time.UTC)

// recordingTransport is a Transport that captures notifications instead of
// putting them on a wire, so a test can assert which qualifying events an
// action raised without standing up a link.
type recordingTransport struct {
	notifications []notification
	connected     bool
	shutdowns     int
	pairing       bluetooth.PairingState
}

type notification struct {
	char bluetooth.CharacteristicType
	data []byte
}

func (r *recordingTransport) SetWriteHandler(bluetooth.WriteHandler)           {}
func (r *recordingTransport) SetReadHandler(bluetooth.ReadHandler)             {}
func (r *recordingTransport) SetConnectionHandler(bluetooth.ConnectionHandler) {}
func (r *recordingTransport) SetCharacteristicData(bluetooth.CharacteristicType, []byte) {
}

func (r *recordingTransport) Notify(charType bluetooth.CharacteristicType, data []byte) error {
	r.notifications = append(r.notifications, notification{char: charType, data: append([]byte(nil), data...)})
	return nil
}
func (r *recordingTransport) IsConnected() bool   { return r.connected }
func (r *recordingTransport) ShutdownConnection() { r.shutdowns++ }
func (r *recordingTransport) GetPairingState() bluetooth.PairingState {
	return r.pairing
}

func (r *recordingTransport) SetPairingState(s bluetooth.PairingState) error {
	r.pairing = s
	return nil
}

// qualifyingEventBits returns the bitmasks notified on the QualifyingEvents
// characteristic, in order.
func (r *recordingTransport) qualifyingEventBits() []uint32 {
	var bits []uint32
	for _, n := range r.notifications {
		if n.char != bluetooth.CharQualifyingEvents || len(n.data) != 4 {
			continue
		}
		bits = append(bits, uint32(n.data[0])|uint32(n.data[1])<<8|uint32(n.data[2])<<16|uint32(n.data[3])<<24)
	}
	return bits
}

// testHarness builds a harness over a real pump state, simulator and router,
// with a recording transport and a frozen clock.
func testHarness(t *testing.T) (*Harness, *state.PumpState, *recordingTransport, http.Handler) {
	t.Helper()

	ps := state.NewPumpState()
	ps.SetClock(state.NewFrozenClock(testInstant))

	transport := &recordingTransport{connected: true, pairing: bluetooth.PairingStatePairStep1}
	// A nil bridge is fine here: no test in this file encodes a message, and
	// handler registration never touches the bridge. The jar-gated tests in
	// actions_jar_test.go cover the encoding side.
	router := handler.NewRouter(nil, ps, transport, protocol.NewTransactionManager(time.Second),
		"go", "", "jar", "", "java", "")

	sim := state.NewSimulator(ps, time.Second)
	sim.Tick()

	log := reqlog.New(64)
	log.SetClock(ps.Now)
	registry := faults.NewRegistry()
	router.SetRequestLog(log)
	router.SetFaultRegistry(registry)

	h := New(Options{
		PumpState: ps,
		Simulator: sim,
		Transport: transport,
		Router:    router,
		Requests:  log,
		Faults:    registry,
	})

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return h, ps, transport, mux
}

// do issues a request against the harness mux and decodes the JSON body.
func do(t *testing.T, mux http.Handler, method, path, body string) (int, map[string]interface{}) {
	t.Helper()

	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	req := httptest.NewRequest(method, path, reader)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var decoded map[string]interface{}
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("%s %s: response is not JSON: %v (%s)", method, path, err, rec.Body.String())
		}
	}
	return rec.Code, decoded
}

// mustDo issues a request and fails unless it succeeded.
func mustDo(t *testing.T, mux http.Handler, method, path, body string) map[string]interface{} {
	t.Helper()

	code, decoded := do(t, mux, method, path, body)
	if code != http.StatusOK {
		t.Fatalf("%s %s = %d: %v", method, path, code, decoded)
	}
	return decoded
}

// ---------------------------------------------------------------------------
// Clock
// ---------------------------------------------------------------------------

func TestClockDefaultsToReal(t *testing.T) {
	ps := state.NewPumpState()
	h := New(Options{PumpState: ps, Requests: reqlog.New(8), Faults: faults.NewRegistry()})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := mustDo(t, mux, http.MethodGet, "/api/clock", "")
	if body["mode"] != ClockModeReal {
		t.Errorf("default clock mode = %v, want %q", body["mode"], ClockModeReal)
	}
	if body["pump_offset_seconds"] != float64(0) {
		t.Errorf("default pump offset = %v, want 0", body["pump_offset_seconds"])
	}
	if body["frozen"] != false {
		t.Errorf("default clock reports frozen = %v", body["frozen"])
	}

	// Advancing without switching to manual mode is refused rather than
	// silently doing nothing.
	code, _ := do(t, mux, http.MethodPost, "/api/clock/advance", `{"seconds":10}`)
	if code != http.StatusConflict {
		t.Errorf("advance on the real clock = %d, want 409", code)
	}
}

func TestClockPutSwitchesToManualAndSkews(t *testing.T) {
	ps := state.NewPumpState()
	h := New(Options{PumpState: ps, Simulator: state.NewSimulator(ps, time.Second)})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body := mustDo(t, mux, http.MethodPut, "/api/clock",
		`{"mode":"manual","now":"2024-03-05T12:00:00Z","frozen":true,"pump_offset_seconds":8}`)

	if body["mode"] != ClockModeManual || body["frozen"] != true {
		t.Fatalf("clock did not switch to a frozen manual clock: %v", body)
	}
	if got := body["now"].(string); !strings.HasPrefix(got, "2024-03-05T12:00:00") {
		t.Errorf("now = %q, want the instant that was set", got)
	}
	if got := body["pump_now"].(string); !strings.HasPrefix(got, "2024-03-05T12:00:08") {
		t.Errorf("pump_now = %q, want now plus the 8 s skew", got)
	}
	// The wire value is local time in the pump's zone, so it is not
	// PumpTimeSeconds (which is UTC) unless the pump is in UTC.
	wantWire := state.PumpTimeSecondsIn(testInstant, ps.GetPumpTimeZone()) + 8
	if got := uint32(body["pump_time_seconds"].(float64)); got != wantWire {
		t.Errorf("pump_time_seconds = %d, want %d", got, wantWire)
	}
	if got := ps.PumpTimeNow(); got != wantWire {
		t.Errorf("the pump state itself reports %d", got)
	}
	if got, want := body["pump_timezone"], ps.GetPumpTimeZone().String(); got != want {
		t.Errorf("pump_timezone = %v, want %q", got, want)
	}
	if got, want := int(body["pump_timezone_offset_seconds"].(float64)), ps.PumpTimeZoneOffsetSeconds(ps.PumpNow()); got != want {
		t.Errorf("pump_timezone_offset_seconds = %d, want %d", got, want)
	}

	// A frozen clock stands still between reads.
	first := mustDo(t, mux, http.MethodGet, "/api/clock", "")["now"]
	time.Sleep(5 * time.Millisecond)
	second := mustDo(t, mux, http.MethodGet, "/api/clock", "")["now"]
	if first != second {
		t.Errorf("a frozen clock moved: %v then %v", first, second)
	}

	// Switching back to the real clock restores the default behavior.
	body = mustDo(t, mux, http.MethodPut, "/api/clock", `{"mode":"real","pump_offset_seconds":0}`)
	if body["mode"] != ClockModeReal {
		t.Errorf("mode = %v after switching back", body["mode"])
	}
	if delta := time.Since(ps.Now()); delta > time.Second {
		t.Errorf("the pump clock is %v behind wall time after switching back", delta)
	}
}

func TestClockAdvanceStepsTheSimulator(t *testing.T) {
	_, ps, _, mux := testHarness(t)

	mustDo(t, mux, http.MethodPut, "/api/clock", `{"mode":"manual","frozen":true,"now":"2024-03-05T12:00:00Z"}`)
	mustDo(t, mux, http.MethodPut, "/api/state", `{"bolus_rate_units_per_second":0.1}`)
	mustDo(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":2}`)

	body := mustDo(t, mux, http.MethodPost, "/api/clock/advance", `{"seconds":10}`)
	if got := body["now"].(string); !strings.HasPrefix(got, "2024-03-05T12:00:10") {
		t.Errorf("now = %q after a 10 s advance", got)
	}

	ps.RLock()
	delivered := ps.Bolus.UnitsDelivered
	ps.RUnlock()
	if delivered < 0.99 || delivered > 1.01 {
		t.Errorf("delivered %.3f units after advancing 10 s at 0.1 U/s, want ~1.0", delivered)
	}

	// Advancing past the end completes the bolus with no wall-clock waiting.
	mustDo(t, mux, http.MethodPost, "/api/clock/advance", `{"seconds":20}`)
	if ps.IsBolusActive() {
		t.Error("the bolus is still active after advancing past its duration")
	}
	record, ok := ps.GetLastBolus()
	if !ok || record.EndReasonID != state.BolusEndReasonCompleted {
		t.Errorf("last bolus record = %+v, ok=%v", record, ok)
	}
}

func TestClockRejectsAnUnknownMode(t *testing.T) {
	_, _, _, mux := testHarness(t)
	code, body := do(t, mux, http.MethodPut, "/api/clock", `{"mode":"sundial"}`)
	if code != http.StatusBadRequest {
		t.Fatalf("PUT with a bad mode = %d, want 400", code)
	}
	if !strings.Contains(body["error"].(string), "sundial") {
		t.Errorf("error did not name the bad mode: %v", body["error"])
	}
}

// ---------------------------------------------------------------------------
// State snapshot and updates
// ---------------------------------------------------------------------------

func TestStateSnapshotReportsThePump(t *testing.T) {
	_, ps, _, mux := testHarness(t)

	body := mustDo(t, mux, http.MethodGet, "/api/state", "")

	identity := body["identity"].(map[string]interface{})
	if identity["serial_number"] != ps.GetSerialNumber() {
		t.Errorf("serial = %v", identity["serial_number"])
	}
	basal := body["basal"].(map[string]interface{})
	if basal["profile_rate"] != ps.GetProfileBasalRate() {
		t.Errorf("profile_rate = %v, want %v", basal["profile_rate"], ps.GetProfileBasalRate())
	}
	if body["last_bolus"] != nil {
		t.Errorf("last_bolus = %v before any bolus finished, want null", body["last_bolus"])
	}
	conn := body["connection"].(map[string]interface{})
	if conn["connected"] != true {
		t.Errorf("connected = %v", conn["connected"])
	}
	if _, ok := body["request_log"]; !ok {
		t.Error("snapshot has no request_log section")
	}
}

func TestStatePutSetsFieldsWithoutRaisingEvents(t *testing.T) {
	_, ps, transport, mux := testHarness(t)

	body := mustDo(t, mux, http.MethodPut, "/api/state",
		`{"reservoir_units":42.5,"battery_percent":17,"basal_rate":1.25,"suspended":true,"iob":3.5,"closed_loop_enabled":true,"cgm_egv":88,"time_since_reset":300}`)

	applied, _ := body["applied"].([]interface{})
	if len(applied) < 7 {
		t.Errorf("applied = %v, want every named field", applied)
	}

	if got := ps.GetReservoirLevel(); got != 42.5 {
		t.Errorf("reservoir = %v", got)
	}
	if got := ps.GetBatteryLevel(); got != 17 {
		t.Errorf("battery = %v", got)
	}
	if got := ps.GetProfileBasalRate(); got != 1.25 {
		t.Errorf("basal rate = %v", got)
	}
	if !ps.IsPumpingSuspended() {
		t.Error("suspended was not applied")
	}
	if !ps.IsClosedLoopEnabled() {
		t.Error("closed_loop_enabled was not applied")
	}
	if got := ps.GetTimeSinceReset(); got != 300 {
		t.Errorf("time_since_reset = %d", got)
	}

	// Staging state is not the pump doing something: no qualifying events.
	if bits := transport.qualifyingEventBits(); len(bits) != 0 {
		t.Errorf("PUT /api/state raised qualifying events %v; only actions should", bits)
	}
}

// ---------------------------------------------------------------------------
// Actions
// ---------------------------------------------------------------------------

func TestBolusStartIsPumpInitiatedAndObservable(t *testing.T) {
	_, ps, transport, mux := testHarness(t)

	body := mustDo(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":2.5,"duration_seconds":50}`)
	bolusID := uint32(body["bolus_id"].(float64))

	if !ps.IsBolusActive() {
		t.Fatal("no bolus is active after bolus/start")
	}
	ps.RLock()
	bolus := *ps.Bolus
	ps.RUnlock()

	if bolus.BolusID != bolusID || bolus.UnitsTotal != 2.5 {
		t.Errorf("bolus = %+v, want id %d and 2.5 units", bolus, bolusID)
	}
	// A pump-initiated bolus must NOT claim it came from the app: a driver
	// uses the source id to tell the two apart.
	if bolus.SourceID != state.BolusSourceQuickBolus {
		t.Errorf("sourceId = %d, want the pump's own %d", bolus.SourceID, state.BolusSourceQuickBolus)
	}
	if got, want := ps.GetBolusRate(), 2.5/50.0; got != want {
		t.Errorf("delivery rate = %v, want %v derived from duration_seconds", got, want)
	}
	if !bolus.StartTime.Equal(testInstant) {
		t.Errorf("start time = %v, want the frozen clock's %v", bolus.StartTime, testInstant)
	}

	if bits := transport.qualifyingEventBits(); len(bits) != 1 || bits[0] != 1024 {
		t.Errorf("qualifying events = %v, want one bolusChange (1024)", bits)
	}
	assertHistoryHas(t, ps, "BolusActivated")

	// Explicit source and id are honored, once the first bolus is out of the way.
	mustDo(t, mux, http.MethodPost, "/api/state/bolus/abort", "")
	body = mustDo(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":1,"source":"remote","bolus_id":77}`)
	if got := uint32(body["bolus_id"].(float64)); got != 77 {
		t.Errorf("bolus_id = %d, want the requested 77", got)
	}
	ps.RLock()
	source := ps.Bolus.SourceID
	ps.RUnlock()
	if source != state.BolusSourceBluetoothRemote {
		t.Errorf("sourceId = %d, want %d for source \"remote\"", source, state.BolusSourceBluetoothRemote)
	}
}

func TestBolusStartRejectsBadRequests(t *testing.T) {
	_, _, _, mux := testHarness(t)

	if code, _ := do(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":0}`); code != http.StatusBadRequest {
		t.Errorf("units=0 = %d, want 400", code)
	}
	if code, _ := do(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":1,"source":"telepathy"}`); code != http.StatusBadRequest {
		t.Errorf("an unknown source = %d, want 400", code)
	}

	mustDo(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":1}`)
	if code, _ := do(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":1}`); code != http.StatusConflict {
		t.Errorf("a second concurrent bolus = %d, want 409", code)
	}
}

func TestBolusStallHoldsDeliveryAndResumeContinuesIt(t *testing.T) {
	_, ps, _, mux := testHarness(t)

	mustDo(t, mux, http.MethodPut, "/api/clock", `{"mode":"manual","frozen":true,"now":"2024-03-05T12:00:00Z"}`)
	mustDo(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":5,"rate":0.1}`)

	mustDo(t, mux, http.MethodPost, "/api/clock/advance", `{"seconds":10}`)
	mustDo(t, mux, http.MethodPost, "/api/state/bolus/stall", "")

	ps.RLock()
	atStall := ps.Bolus.UnitsDelivered
	ps.RUnlock()

	mustDo(t, mux, http.MethodPost, "/api/clock/advance", `{"seconds":600}`)

	ps.RLock()
	afterStall := ps.Bolus.UnitsDelivered
	stalled := ps.Bolus.Stalled
	active := ps.Bolus.Active
	ps.RUnlock()

	if !stalled || !active {
		t.Fatalf("stalled=%v active=%v; a stalled bolus stays in progress", stalled, active)
	}
	if afterStall != atStall {
		t.Errorf("a stalled bolus delivered %.3f -> %.3f units over 10 minutes", atStall, afterStall)
	}

	mustDo(t, mux, http.MethodPost, "/api/state/bolus/resume", "")
	mustDo(t, mux, http.MethodPost, "/api/clock/advance", `{"seconds":10}`)

	ps.RLock()
	afterResume := ps.Bolus.UnitsDelivered
	ps.RUnlock()
	if delta := afterResume - atStall; delta < 0.99 || delta > 1.01 {
		t.Errorf("after resuming, 10 s delivered %.3f units, want ~1.0 (no catch-up for the stall)", delta)
	}
}

func TestBolusAbortAndCompleteRecordTheLastBolus(t *testing.T) {
	_, ps, transport, mux := testHarness(t)

	mustDo(t, mux, http.MethodPut, "/api/clock", `{"mode":"manual","frozen":true,"now":"2024-03-05T12:00:00Z"}`)
	body := mustDo(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":4,"rate":0.1}`)
	bolusID := uint32(body["bolus_id"].(float64))
	mustDo(t, mux, http.MethodPost, "/api/clock/advance", `{"seconds":10}`)
	mustDo(t, mux, http.MethodPost, "/api/state/bolus/abort", "")

	record, ok := ps.GetLastBolus()
	if !ok {
		t.Fatal("no last-bolus record after an abort")
	}
	if record.BolusID != bolusID {
		t.Errorf("last bolus id = %d, want the aborted bolus %d", record.BolusID, bolusID)
	}
	if record.EndReasonID != state.BolusEndReasonStopped {
		t.Errorf("end reason = %d, want stopped (%d)", record.EndReasonID, state.BolusEndReasonStopped)
	}
	if record.DeliveredUnits < 0.99 || record.DeliveredUnits > 1.01 {
		t.Errorf("delivered = %.3f, want the ~1.0 that actually went in", record.DeliveredUnits)
	}
	if record.RequestedUnits != 4 {
		t.Errorf("requested = %.3f, want 4", record.RequestedUnits)
	}
	if !record.EndTime.Equal(testInstant.Add(10 * time.Second)) {
		t.Errorf("end time = %v, want the pump clock's %v", record.EndTime, testInstant.Add(10*time.Second))
	}
	assertHistoryHas(t, ps, "BolusCompleted")

	// A completion delivers the full requested volume.
	mustDo(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":3,"rate":0.1}`)
	mustDo(t, mux, http.MethodPost, "/api/state/bolus/complete", "")

	record, _ = ps.GetLastBolus()
	if record.DeliveredUnits != 3 || record.EndReasonID != state.BolusEndReasonCompleted {
		t.Errorf("completed record = %+v, want 3 units and end reason %d", record, state.BolusEndReasonCompleted)
	}

	if bits := transport.qualifyingEventBits(); len(bits) != 4 {
		t.Errorf("qualifying events = %v, want start/abort/start/complete", bits)
	}

	if code, _ := do(t, mux, http.MethodPost, "/api/state/bolus/abort", ""); code != http.StatusConflict {
		t.Errorf("aborting with no bolus running = %d, want 409", code)
	}
}

func TestTempBasalStartAndStop(t *testing.T) {
	_, ps, transport, mux := testHarness(t)

	mustDo(t, mux, http.MethodPut, "/api/state", `{"basal_rate":1.0}`)
	body := mustDo(t, mux, http.MethodPost, "/api/state/tempbasal/start", `{"percent":150,"duration_minutes":30}`)

	temp := ps.GetTempRate()
	if !temp.Active || temp.Percent != 150 {
		t.Fatalf("temp rate = %+v, want an active 150%%", temp)
	}
	if temp.Rate != 1.5 {
		t.Errorf("temp rate = %v U/hr, want 150%% of the 1.0 profile rate", temp.Rate)
	}
	if !temp.StartTime.Equal(testInstant) || !temp.EndTime.Equal(testInstant.Add(30*time.Minute)) {
		t.Errorf("temp window = %v..%v", temp.StartTime, temp.EndTime)
	}
	if got := int(body["temp_rate_id"].(float64)); got != temp.TempRateID || got == 0 {
		t.Errorf("temp_rate_id = %d, state has %d", got, temp.TempRateID)
	}
	// A percentage applies to the PROFILE rate, which must be left intact.
	if got := ps.GetProfileBasalRate(); got != 1.0 {
		t.Errorf("profile rate = %v after a temp rate, want 1.0", got)
	}
	assertHistoryHas(t, ps, "TempRateActivated")

	mustDo(t, mux, http.MethodPost, "/api/state/tempbasal/stop", "")
	if ps.GetTempRate().Active {
		t.Error("temp rate still active after stop")
	}
	if got := ps.GetBasalRate(); got != 1.0 {
		t.Errorf("basal rate = %v after stopping the temp rate, want the profile 1.0", got)
	}
	assertHistoryHas(t, ps, "TempRateCompleted")

	if bits := transport.qualifyingEventBits(); len(bits) != 2 || bits[0] != 512 || bits[1] != 512 {
		t.Errorf("qualifying events = %v, want two basalChange (512)", bits)
	}

	if code, _ := do(t, mux, http.MethodPost, "/api/state/tempbasal/stop", ""); code != http.StatusConflict {
		t.Errorf("stopping an absent temp rate = %d, want 409", code)
	}
}

func TestTempBasalAcceptsAnAbsoluteRate(t *testing.T) {
	_, ps, _, mux := testHarness(t)

	mustDo(t, mux, http.MethodPut, "/api/state", `{"basal_rate":0.8}`)
	mustDo(t, mux, http.MethodPost, "/api/state/tempbasal/start", `{"rate":1.2,"duration_minutes":30}`)

	temp := ps.GetTempRate()
	if temp.Rate != 1.2 {
		t.Errorf("temp rate = %v, want the absolute 1.2 U/hr", temp.Rate)
	}
	if temp.Percent != 150 {
		t.Errorf("percent = %d, want 150 derived from 1.2/0.8", temp.Percent)
	}
}

func TestSuspendStopsBolusAndTempRateAndRaisesEverything(t *testing.T) {
	_, ps, transport, mux := testHarness(t)

	mustDo(t, mux, http.MethodPut, "/api/clock", `{"mode":"manual","frozen":true,"now":"2024-03-05T12:00:00Z"}`)
	mustDo(t, mux, http.MethodPost, "/api/state/tempbasal/start", `{"percent":150,"duration_minutes":30}`)
	body := mustDo(t, mux, http.MethodPost, "/api/state/bolus/start", `{"units":5,"rate":0.1}`)
	bolusID := uint32(body["bolus_id"].(float64))
	mustDo(t, mux, http.MethodPost, "/api/clock/advance", `{"seconds":10}`)

	mustDo(t, mux, http.MethodPost, "/api/state/suspend", `{"reason":"occlusion"}`)

	if !ps.IsPumpingSuspended() {
		t.Fatal("the pump is not suspended")
	}
	if ps.IsBolusActive() {
		t.Error("the bolus survived the suspend; a real stop halts delivery")
	}
	if ps.GetTempRate().Active {
		t.Error("the temp rate survived the suspend")
	}

	record, ok := ps.GetLastBolus()
	if !ok || record.BolusID != bolusID || record.EndReasonID != state.BolusEndReasonStopped {
		t.Errorf("last bolus = %+v (ok=%v), want the stopped bolus %d", record, ok, bolusID)
	}
	if record.DeliveredUnits < 0.99 || record.DeliveredUnits > 1.01 {
		t.Errorf("partial delivery = %.3f, want the ~1.0 that went in before the stop", record.DeliveredUnits)
	}

	ps.RLock()
	alerts := len(ps.ActiveAlerts)
	alertType := state.AlertType(-1)
	if alerts > 0 {
		alertType = ps.ActiveAlerts[0].Type
	}
	reason := ps.SuspendReason()
	ps.RUnlock()

	if alerts != 1 || alertType != state.AlertOcclusion {
		t.Errorf("alerts = %d of type %v, want one occlusion alarm", alerts, alertType)
	}
	if reason != SuspendReasonOcclusion {
		t.Errorf("suspend reason = %q", reason)
	}

	for _, want := range []string{"BolusCompleted", "TempRateCompleted", "PumpingSuspended", "AlarmActivated"} {
		assertHistoryHas(t, ps, want)
	}

	bits := transport.qualifyingEventBits()
	assertHasBit(t, bits, 1024, "bolusChange")
	assertHasBit(t, bits, 512, "basalChange")
	assertHasBit(t, bits, 64, "pumpSuspend")
	assertHasBit(t, bits, 1, "alert")

	if code, _ := do(t, mux, http.MethodPost, "/api/state/suspend", ""); code != http.StatusConflict {
		t.Errorf("suspending twice = %d, want 409", code)
	}
}

func TestSuspendByUserRaisesNoAlarm(t *testing.T) {
	_, ps, transport, mux := testHarness(t)

	mustDo(t, mux, http.MethodPost, "/api/state/suspend", `{"reason":"user"}`)

	ps.RLock()
	alerts := len(ps.ActiveAlerts)
	ps.RUnlock()
	if alerts != 0 {
		t.Errorf("a user suspend raised %d alarm(s)", alerts)
	}
	bits := transport.qualifyingEventBits()
	assertHasBit(t, bits, 64, "pumpSuspend")
	for _, b := range bits {
		if b == 1 {
			t.Error("a user suspend raised an alert qualifying event")
		}
	}

	if code, _ := do(t, mux, http.MethodPost, "/api/state/suspend", `{"reason":"sulking"}`); code != http.StatusBadRequest {
		t.Errorf("an unknown reason = %d, want 400", code)
	}
}

func TestResumeClearsAlarmsAndRaisesResume(t *testing.T) {
	_, ps, transport, mux := testHarness(t)

	mustDo(t, mux, http.MethodPost, "/api/state/suspend", `{"reason":"occlusion"}`)
	mustDo(t, mux, http.MethodPost, "/api/state/resume", "")

	if ps.IsPumpingSuspended() {
		t.Fatal("still suspended after resume")
	}
	ps.RLock()
	alerts := len(ps.ActiveAlerts)
	reason := ps.SuspendReason()
	ps.RUnlock()
	if alerts != 0 {
		t.Errorf("%d alarm(s) survived the resume", alerts)
	}
	if reason != "" {
		t.Errorf("suspend reason = %q after resume, want empty", reason)
	}
	assertHistoryHas(t, ps, "PumpingResumed")

	bits := transport.qualifyingEventBits()
	assertHasBit(t, bits, 128, "pumpResume")
	assertHasBit(t, bits, 512, "basalChange")

	if code, _ := do(t, mux, http.MethodPost, "/api/state/resume", ""); code != http.StatusConflict {
		t.Errorf("resuming an unsuspended pump = %d, want 409", code)
	}
}

func TestResumeCanKeepAlarms(t *testing.T) {
	_, ps, _, mux := testHarness(t)

	mustDo(t, mux, http.MethodPost, "/api/state/suspend", `{"reason":"alarm"}`)
	mustDo(t, mux, http.MethodPost, "/api/state/resume", `{"clear_alarms":false}`)

	ps.RLock()
	alerts := len(ps.ActiveAlerts)
	ps.RUnlock()
	if alerts != 1 {
		t.Errorf("alerts = %d, want the alarm kept when clear_alarms is false", alerts)
	}
}

func TestHistoryAppendBackdates(t *testing.T) {
	_, ps, _, mux := testHarness(t)

	body := mustDo(t, mux, http.MethodPost, "/api/state/history/append",
		`{"type":"BolusCompleted","seconds_ago":3600,"data":{"bolusId":9}}`)

	if got := int(body["type_id"].(float64)); got != state.HistoryBolusCompleted {
		t.Errorf("type_id = %d, want %d looked up from the name", got, state.HistoryBolusCompleted)
	}
	seq := uint32(body["sequence"].(float64))

	entries := ps.GetHistoryLogEntries(seq, seq)
	if len(entries) != 1 {
		t.Fatalf("appended record not found at sequence %d", seq)
	}
	want := testInstant.Add(-time.Hour)
	if !entries[0].Timestamp.Equal(want) {
		t.Errorf("record time = %v, want the backdated %v", entries[0].Timestamp, want)
	}
	if wantWire := ps.PumpTimeFor(want); entries[0].PumpTime != wantWire {
		t.Errorf("record pump time = %d, want %d", entries[0].PumpTime, wantWire)
	}

	if code, _ := do(t, mux, http.MethodPost, "/api/state/history/append", `{}`); code != http.StatusBadRequest {
		t.Errorf("append with no type = %d, want 400", code)
	}
}

func TestQualifyingEventEndpointSendsARawBitmask(t *testing.T) {
	_, _, transport, mux := testHarness(t)

	mustDo(t, mux, http.MethodPost, "/api/state/qualifyingevent", `{"bitmask":262144}`)
	bits := transport.qualifyingEventBits()
	if len(bits) != 1 || bits[0] != 262144 {
		t.Errorf("qualifying events = %v, want the requested 262144", bits)
	}
}

func TestUnknownActionAndWrongMethod(t *testing.T) {
	_, _, _, mux := testHarness(t)

	if code, _ := do(t, mux, http.MethodPost, "/api/state/levitate", "{}"); code != http.StatusNotFound {
		t.Errorf("an unknown action = %d, want 404", code)
	}
	if code, _ := do(t, mux, http.MethodGet, "/api/state/suspend", ""); code != http.StatusMethodNotAllowed {
		t.Errorf("GET on an action = %d, want 405", code)
	}
}

// ---------------------------------------------------------------------------
// Request log
// ---------------------------------------------------------------------------

func TestLogEndpointPagesAndClears(t *testing.T) {
	h, _, _, mux := testHarness(t)

	h.requests.RecordRequest("Control", "InitiateBolusRequest", 64, 1, nil, []string{"aa"})
	h.requests.RecordResponse("Control", "InitiateBolusResponse", 65, 1, []string{"bb"}, 1, "", "")

	body := mustDo(t, mux, http.MethodGet, "/api/log", "")
	entries := body["entries"].([]interface{})
	if len(entries) != 2 {
		t.Fatalf("log has %d entries, want 2", len(entries))
	}
	first := entries[0].(map[string]interface{})
	if first["message"] != "InitiateBolusRequest" || first["kind"] != "request" {
		t.Errorf("first entry = %v", first)
	}
	lastSeq := int(body["last_seq"].(float64))

	// since= returns only what is newer.
	body = mustDo(t, mux, http.MethodGet, "/api/log?since=1", "")
	if got := len(body["entries"].([]interface{})); got != 1 {
		t.Errorf("since=1 returned %d entries, want 1", got)
	}
	body = mustDo(t, mux, http.MethodGet, "/api/log?since=99", "")
	if got := len(body["entries"].([]interface{})); got != 0 {
		t.Errorf("since=99 returned %d entries, want none", got)
	}

	if code, _ := do(t, mux, http.MethodGet, "/api/log?since=abc", ""); code != http.StatusBadRequest {
		t.Errorf("a non-numeric since = %d, want 400", code)
	}

	mustDo(t, mux, http.MethodDelete, "/api/log", "")
	body = mustDo(t, mux, http.MethodGet, "/api/log", "")
	if got := len(body["entries"].([]interface{})); got != 0 {
		t.Errorf("log still holds %d entries after DELETE", got)
	}
	if got := int(body["last_seq"].(float64)); got != lastSeq {
		t.Errorf("last_seq reset to %d; sequence numbers must keep moving forward", got)
	}
}

// ---------------------------------------------------------------------------
// Fault endpoints
// ---------------------------------------------------------------------------

func TestFaultsArmListAndClear(t *testing.T) {
	h, _, _, mux := testHarness(t)

	body := mustDo(t, mux, http.MethodPost, "/api/faults",
		`{"kind":"drop_response","message":"SetTempRateRequest"}`)
	fault := body["fault"].(map[string]interface{})
	id := int(fault["id"].(float64))
	if fault["kind"] != "drop_response" || int(fault["count"].(float64)) != 1 {
		t.Errorf("armed fault = %v", fault)
	}

	body = mustDo(t, mux, http.MethodGet, "/api/faults", "")
	if got := len(body["faults"].([]interface{})); got != 1 {
		t.Fatalf("listed %d faults, want 1", got)
	}
	if h.registry.Peek(faults.Target{Message: "SetTempRateResponse"}, faults.KindDropResponse) == nil {
		t.Error("the armed fault does not match its own message")
	}

	mustDo(t, mux, http.MethodDelete, "/api/faults/"+itoa(id), "")
	if got := len(h.registry.List()); got != 0 {
		t.Errorf("%d faults survived the delete", got)
	}

	mustDo(t, mux, http.MethodPost, "/api/faults", `{"kind":"delay_response","delay_ms":5,"every":true}`)
	mustDo(t, mux, http.MethodPost, "/api/faults", `{"kind":"disconnect","after":"partial_response","fragments_sent":1}`)
	body = mustDo(t, mux, http.MethodDelete, "/api/faults", "")
	if got := int(body["cleared"].(float64)); got != 2 {
		t.Errorf("cleared %d faults, want 2", got)
	}
}

func TestFaultsRejectBadDefinitions(t *testing.T) {
	_, _, _, mux := testHarness(t)

	for _, body := range []string{`{}`, `{"kind":"explode"}`, `{"kind":"delay_response"}`} {
		if code, _ := do(t, mux, http.MethodPost, "/api/faults", body); code != http.StatusBadRequest {
			t.Errorf("POST %s = %d, want 400", body, code)
		}
	}
	if code, _ := do(t, mux, http.MethodDelete, "/api/faults/nope", ""); code != http.StatusBadRequest {
		t.Errorf("deleting a non-numeric id = %d, want 400", code)
	}
	if code, _ := do(t, mux, http.MethodDelete, "/api/faults/999", ""); code != http.StatusNotFound {
		t.Errorf("deleting an unknown id = %d, want 404", code)
	}
}

func TestRadioFaultNeedsACapableTransport(t *testing.T) {
	_, _, _, mux := testHarness(t)

	// The recording transport is not a RadioController, so the harness says so
	// rather than silently doing nothing.
	if code, _ := do(t, mux, http.MethodPost, "/api/faults", `{"kind":"radio_off"}`); code != http.StatusNotImplemented {
		t.Errorf("radio_off on a plain transport = %d, want 501", code)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func assertHistoryHas(t *testing.T, ps *state.PumpState, entryType string) {
	t.Helper()

	for _, e := range ps.GetHistoryLogEntries(0, 1<<30) {
		if e.Type == entryType {
			return
		}
	}
	t.Errorf("no %s record in the history log", entryType)
}

func assertHasBit(t *testing.T, bits []uint32, want uint32, name string) {
	t.Helper()

	for _, b := range bits {
		if b == want {
			return
		}
	}
	t.Errorf("qualifying events %v do not include %s (%d)", bits, name, want)
}

func itoa(i int) string { return strconv.Itoa(i) }

// TestPairingCodeUpdateReachesObservers is the regression this covers: only the
// websocket setPairingCode command told the pumpX2 bridge about a new pairing
// code, so a code set through PUT/PATCH /api/state changed what the pump
// reported but not the JPAKE password the bridge handed to cliparser. A harness
// that set a code and then paired authenticated against the OLD code.
//
// Both routes now go through PumpState.SetPairingCode, which notifies every
// observer -- main() registers the bridge as one.
func TestPairingCodeUpdateReachesObservers(t *testing.T) {
	_, ps, _, mux := testHarness(t)

	var observed []string
	ps.OnPairingCodeChange(func(code string) { observed = append(observed, code) })

	// A cached long-term key from an earlier pairing must not survive a code
	// change either: it is derived from the old password.
	ps.SetLongTermKey([]byte{1, 2, 3, 4})

	mustDo(t, mux, http.MethodPut, "/api/state", `{"pairing_code":"654321"}`)

	if got := ps.GetPairingCode(); got != "654321" {
		t.Errorf("pairing code = %q, want 654321", got)
	}
	if len(observed) != 1 || observed[0] != "654321" {
		t.Errorf("observers saw %v, want exactly one notification of 654321", observed)
	}
	if key := ps.GetLongTermKey(); key != nil {
		t.Errorf("long-term key = %x, want it cleared by the pairing code change", key)
	}

	// The snapshot must agree, so a harness can read back what it set.
	snapshot := mustDo(t, mux, http.MethodGet, "/api/state", "")
	auth, _ := snapshot["auth"].(map[string]interface{})
	if auth["pairing_code"] != "654321" {
		t.Errorf("snapshot pairing_code = %v, want 654321", auth["pairing_code"])
	}

	// PATCH is the same path.
	mustDo(t, mux, http.MethodPatch, "/api/state", `{"pairing_code":"111111"}`)
	if len(observed) != 2 || observed[1] != "111111" {
		t.Errorf("observers saw %v, want a second notification of 111111", observed)
	}
}
