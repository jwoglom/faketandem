package harness

import (
	"net/http"
	"testing"

	"github.com/jwoglom/faketandem/pkg/handler"
)

func cgmField(t *testing.T, body map[string]interface{}, key string) interface{} {
	t.Helper()
	if cgm, ok := body["cgm"].(map[string]interface{}); ok {
		body = cgm
	}
	v, ok := body[key]
	if !ok {
		t.Fatalf("no %q in %v", key, body)
	}
	return v
}

func countBits(bits []uint32, want uint32) int {
	n := 0
	for _, b := range bits {
		if b&want != 0 {
			n++
		}
	}
	return n
}

func TestTheCGMAPIWalksAG7SessionThroughItsLifecycle(t *testing.T) {
	h, _, transport, mux := testHarness(t)
	h.simulator.SetEventNotifier(h.router.GetQualifyingEventsNotifier())

	body := mustDo(t, mux, http.MethodPost, "/api/cgm/reset", `{"sensor_type":"g7"}`)
	if cgmField(t, body, "phase") != "stopped" || cgmField(t, body, "sensor_type_name") != "g7" {
		t.Fatalf("reset = %v", body)
	}
	mustDo(t, mux, http.MethodPut, "/api/cgm", `{"auto":false,"g7_pairing_code":"0042","egv":95}`)

	for _, want := range []string{"start_pending", "warmup", "active", "grace"} {
		body = mustDo(t, mux, http.MethodPost, "/api/cgm/advance", `{"to":"next"}`)
		if got := cgmField(t, body, "phase"); got != want {
			t.Fatalf("advance next = %v, want %s", got, want)
		}
	}
	if cgmField(t, body, "grace_period") != true || cgmField(t, body, "session_state").(float64) != 2 {
		t.Fatalf("grace = %v", body)
	}
	alerts := cgmField(t, body, "alerts").([]interface{})
	if len(alerts) != 1 || alerts[0] != "GRACE_PERIOD_START" {
		t.Fatalf("grace alerts = %v", alerts)
	}

	body = mustDo(t, mux, http.MethodPost, "/api/cgm/advance", `{"to":"active","session_age_seconds":3600}`)
	if cgmField(t, body, "current_egv").(float64) != 95 || cgmField(t, body, "session_remaining_seconds").(float64) != 10*24*3600-3600 {
		t.Fatalf("active an hour in = %v", body)
	}
	if code := cgmField(t, body, "g7_pairing_code"); code.(float64) != 42 {
		t.Fatalf("g7_pairing_code = %v", code)
	}

	bits := transport.qualifyingEventBits()
	if n := countBits(bits, handler.QECGMChange); n != 6 {
		t.Errorf("CGM_CHANGE sent %d times for reset, start pending, warmup, active, grace and active: %v", n, bits)
	}
	if n := countBits(bits, handler.QECGMAlert); n != 1 {
		t.Errorf("CGM_ALERT sent %d times, want once for the grace period: %v", n, bits)
	}

	state := mustDo(t, mux, http.MethodGet, "/api/state", "")
	if cgmField(t, state["cgm"].(map[string]interface{}), "phase") != "active" {
		t.Errorf("/api/state's cgm = %v", state["cgm"])
	}
}

func TestTheCGMAPITimesASessionOnThePumpClock(t *testing.T) {
	_, _, _, mux := testHarness(t)
	mustDo(t, mux, http.MethodPut, "/api/clock", `{"mode":"manual","frozen":true,"now":"2024-03-05T12:00:00Z"}`)

	mustDo(t, mux, http.MethodPost, "/api/cgm/reset", `{"sensor_type":"g7"}`)
	mustDo(t, mux, http.MethodPut, "/api/cgm",
		`{"g7_pairing_code":1234,"timing":{"connect_seconds":10,"warmup_seconds":60}}`)
	mustDo(t, mux, http.MethodPost, "/api/cgm/advance", `{"to":"start_pending"}`)
	mustDo(t, mux, http.MethodPost, "/api/clock/advance", `{"seconds":10}`)
	if got := cgmField(t, mustDo(t, mux, http.MethodGet, "/api/cgm", ""), "phase"); got != "warmup" {
		t.Fatalf("10 s after the start: %v", got)
	}
	mustDo(t, mux, http.MethodPost, "/api/clock/advance", `{"seconds":60}`)
	body := mustDo(t, mux, http.MethodGet, "/api/cgm", "")
	if cgmField(t, body, "phase") != "active" || cgmField(t, body, "egv_status").(float64) != 1 {
		t.Fatalf("after warmup: %v", body)
	}
}

func TestTheCGMAPIPairsOnlyTheSensorWorn(t *testing.T) {
	_, _, _, mux := testHarness(t)
	mustDo(t, mux, http.MethodPut, "/api/clock", `{"mode":"manual","frozen":true,"now":"2024-03-05T12:00:00Z"}`)

	mustDo(t, mux, http.MethodPost, "/api/cgm/reset", `{"sensor_type":"g6"}`)
	mustDo(t, mux, http.MethodPut, "/api/cgm",
		`{"transmitter_id":"8gxyz1","body":{"transmitter_id":"8GABC2"},"timing":{"connect_seconds":5}}`)
	mustDo(t, mux, http.MethodPost, "/api/cgm/advance", `{"to":"start_pending"}`)
	mustDo(t, mux, http.MethodPost, "/api/clock/advance", `{"seconds":5}`)
	body := mustDo(t, mux, http.MethodGet, "/api/cgm", "")
	if cgmField(t, body, "phase") != "stopped" || cgmField(t, body, "end_reason") != "not_found" {
		t.Fatalf("a pairing with another transmitter: %v", body)
	}
}

func TestTheCGMAPIRejectsWhatItCannotApply(t *testing.T) {
	_, ps, _, mux := testHarness(t)

	for _, c := range []struct{ method, path, body string }{
		{http.MethodPut, "/api/cgm", `{"sensor_type":"g9"}`},
		{http.MethodPut, "/api/cgm", `{"egv":80,"refuse":{"bogus":1}}`},
		{http.MethodPut, "/api/cgm", `{"g7_pairing_code":12345}`},
		{http.MethodPut, "/api/cgm", `{"transmitter_battery":"flat"}`},
		{http.MethodPost, "/api/cgm/advance", `{"to":"sideways"}`},
		{http.MethodPost, "/api/cgm/frobnicate", ``},
	} {
		if code, _ := do(t, mux, c.method, c.path, c.body); code == http.StatusOK {
			t.Errorf("%s %s %s was accepted", c.method, c.path, c.body)
		}
	}
	if ps.CGM().EGV == 80 {
		t.Error("a rejected PUT applied part of itself")
	}

	body := mustDo(t, mux, http.MethodPut, "/api/cgm", `{"refuse":{"g7_code":2,"stop":0}}`)
	if refuse := cgmField(t, body, "refuse").(map[string]interface{}); len(refuse) != 1 || refuse["SetDexcomG7PairingCodeRequest"] != 2.0 {
		t.Fatalf("refuse = %v", refuse)
	}
}
