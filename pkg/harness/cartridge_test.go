package harness

import (
	"net/http"
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/state"
)

func authenticatedHarness(t *testing.T) (*Harness, *state.PumpState, http.Handler) {
	t.Helper()
	h, ps, _, mux := testHarness(t)
	ps.SetPumpingSuspended(true)
	ps.Lock()
	ps.AuthKey = []byte("0123456789abcdef0123456789abcdef")
	ps.Unlock()
	return h, ps, mux
}

// streamedCount counts the stream messages named message in the request log.
func streamedCount(h *Harness, message string) int {
	n := 0
	for _, e := range h.requests.All() {
		if e.Message == message {
			n++
		}
	}
	return n
}

func waitForStreamed(t *testing.T, h *Harness, message string, n int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for streamedCount(h, message) < n {
		if time.Now().After(deadline) {
			t.Fatalf("%d %s streamed, want %d", streamedCount(h, message), message, n)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func cartridgeOf(body map[string]interface{}) map[string]interface{} {
	return body["state"].(map[string]interface{})["cartridge"].(map[string]interface{})
}

func TestCartridgeSettingsAreStagedAndReported(t *testing.T) {
	_, _, mux := authenticatedHarness(t)

	body := mustDo(t, mux, http.MethodPut, "/api/state", `{"cartridge_ready_seconds":1.5,"cartridge_detect_seconds":0,
		"cartridge_detect_on_insert":true,"fill_tubing_units_per_second":2,"max_fill_seconds":9,"fill_cannula_stream_lost":true}`)
	settings := cartridgeOf(body)["settings"].(map[string]interface{})
	want := map[string]interface{}{"ready_seconds": 1.5, "detect_seconds": 0.0, "detect_on_insert": true,
		"fill_tubing_units_per_second": 2.0, "max_fill_seconds": 9.0, "fill_cannula_stream_lost": true}
	for k, v := range want {
		if settings[k] != v {
			t.Errorf("settings.%s = %v, want %v", k, settings[k], v)
		}
	}

	for _, bad := range []string{`{"cartridge_ready_seconds":-1}`, `{"max_fill_seconds":0}`, `{"fill_tubing_units_per_second":-2}`} {
		if code, _ := do(t, mux, http.MethodPut, "/api/state", bad); code != http.StatusBadRequest {
			t.Errorf("%s answered %d, want 400", bad, code)
		}
	}
}

func TestCartridgeLoadedIsStaged(t *testing.T) {
	_, _, mux := authenticatedHarness(t)
	body := mustDo(t, mux, http.MethodPut, "/api/state", `{"cartridge_loaded":true}`)
	if c := cartridgeOf(body); c["load_state"] != "load_cartridge" || c["loading_active"] != true {
		t.Errorf("cartridge %v, want load_cartridge active", c)
	}
}

func TestCartridgeReadyStreamsReadyToChange(t *testing.T) {
	h, ps, mux := authenticatedHarness(t)
	if code, _ := do(t, mux, http.MethodPost, "/api/state/cartridge/ready", `{}`); code != http.StatusConflict {
		t.Errorf("ready outside change cartridge mode answered %d, want 409", code)
	}

	if !ps.EnterChangeCartridge(0x21) {
		t.Fatal("change cartridge refused")
	}
	body := mustDo(t, mux, http.MethodPost, "/api/state/cartridge/ready", `{}`)
	if c := cartridgeOf(body); c["ready_to_change"] != true || c["load_state"] != "change_cartridge" {
		t.Errorf("cartridge %v, want ready in change_cartridge", c)
	}
	if got := streamedCount(h, "EnterChangeCartridgeModeStateStreamResponse"); got != 1 {
		t.Errorf("streamed %d EnterChangeCartridgeModeStateStreamResponses, want 1", got)
	}
}

func TestCartridgeInsertChecksTheCartridgeWithDetectOnInsert(t *testing.T) {
	h, ps, mux := authenticatedHarness(t)
	mustDo(t, mux, http.MethodPut, "/api/state", `{"cartridge_detect_on_insert":true,"cartridge_detect_seconds":0.1}`)

	for _, bad := range []string{`{}`, `{"units":-5}`, `{"units":5000}`} {
		if code, _ := do(t, mux, http.MethodPost, "/api/state/cartridge/insert", bad); code != http.StatusBadRequest {
			t.Errorf("%s answered %d, want 400", bad, code)
		}
	}
	if code, _ := do(t, mux, http.MethodPost, "/api/state/cartridge/insert", `{"units":300}`); code != http.StatusConflict {
		t.Errorf("insert outside change cartridge mode answered %d, want 409", code)
	}
	ps.EnterChangeCartridge(0x21)
	if code, _ := do(t, mux, http.MethodPost, "/api/state/cartridge/insert", `{"units":300}`); code != http.StatusConflict {
		t.Errorf("insert before the pump retracted answered %d, want 409", code)
	}

	mustDo(t, mux, http.MethodPost, "/api/state/cartridge/ready", `{}`)
	mustDo(t, mux, http.MethodPost, "/api/state/cartridge/insert", `{"units":300}`)
	waitForStreamed(t, h, "DetectingCartridgeStateStreamResponse", 6)

	body := mustDo(t, mux, http.MethodGet, "/api/state", "")
	c := body["cartridge"].(map[string]interface{})
	if c["loaded"] != true || c["detecting"] != false || c["load_state"] != "change_cartridge" {
		t.Errorf("cartridge %v, want checked and still in change mode", c)
	}
	if units := body["insulin"].(map[string]interface{})["reservoir_units"]; units != 300.0 {
		t.Errorf("reservoir %v, want 300", units)
	}
	if got := streamedCount(h, "QualifyingEvents(0x00040000)"); got != 1 {
		t.Errorf("%d remainingInsulin qualifying events, want 1", got)
	}
}

func TestFillTubingHoldDoesNotReleaseAgainAfterThePumpStopped(t *testing.T) {
	h, ps, mux := authenticatedHarness(t)
	mustDo(t, mux, http.MethodPut, "/api/state", `{"max_fill_seconds":0.05}`)
	if !ps.EnterFillTubing(0x76) {
		t.Fatal("fill tubing refused")
	}
	mustDo(t, mux, http.MethodPost, "/api/state/filltubing/button", `{"hold_seconds":0.3}`)
	waitForStreamed(t, h, "FillTubingStateStreamResponse", 2)
	time.Sleep(500 * time.Millisecond)

	if got := streamedCount(h, "FillTubingStateStreamResponse"); got != 2 {
		t.Errorf("streamed %d button states, want the press and the pump's release", got)
	}
}
