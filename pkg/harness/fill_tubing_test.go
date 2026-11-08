package harness

import (
	"net/http"
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/state"
)

func fillTubingHarness(t *testing.T) (*Harness, *state.PumpState, *recordingTransport, http.Handler) {
	t.Helper()
	h, ps, transport, mux := testHarness(t)
	ps.Lock()
	ps.AuthKey = []byte("0123456789abcdef0123456789abcdef")
	ps.Unlock()
	if !ps.EnterFillTubing(0x76) {
		t.Fatal("fill tubing mode refused")
	}
	return h, ps, transport, mux
}

// fillTubingStreamed counts the FillTubingStateStreamResponses the request log holds. Reading the
// log, which is locked, also orders the transport's notifications before the caller's reads.
func fillTubingStreamed(h *Harness) int {
	n := 0
	for _, e := range h.requests.All() {
		if e.Message == "FillTubingStateStreamResponse" {
			n++
		}
	}
	return n
}

func TestFillTubingButtonPressAndRelease(t *testing.T) {
	h, ps, transport, mux := fillTubingHarness(t)

	body := mustDo(t, mux, http.MethodPost, "/api/state/filltubing/button", `{"down":true}`)
	if body["button_down"] != true || body["state"].(map[string]interface{})["fill_tubing_button_down"] != true {
		t.Errorf("pressing answered %v", body)
	}
	if !ps.FillTubingButtonDown() {
		t.Error("the pump does not read the button down")
	}

	mustDo(t, mux, http.MethodPost, "/api/state/filltubing/button", `{"down":false}`)
	if ps.FillTubingButtonDown() {
		t.Error("the pump still reads the button down after release")
	}

	if got := fillTubingStreamed(h); got != 2 {
		t.Errorf("streamed %d FillTubingStateStreamResponses, want 2", got)
	}
	for _, n := range transport.notifications {
		if n.char != bluetooth.CharControlStream && n.char != bluetooth.CharQualifyingEvents {
			t.Errorf("notified on %s", n.char)
		}
	}
}

func TestFillTubingButtonHoldReleasesItself(t *testing.T) {
	h, ps, _, mux := fillTubingHarness(t)

	mustDo(t, mux, http.MethodPost, "/api/state/filltubing/button", `{"hold_seconds":0.05}`)
	if !ps.FillTubingButtonDown() {
		t.Fatal("a hold did not press the button")
	}

	deadline := time.Now().Add(5 * time.Second)
	for fillTubingStreamed(h) < 2 {
		if time.Now().After(deadline) {
			t.Fatal("the hold never released the button")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if ps.FillTubingButtonDown() {
		t.Error("the button reads down after the hold ran out")
	}
}

func TestFillTubingButtonPressCancelsAPendingRelease(t *testing.T) {
	h, ps, _, mux := fillTubingHarness(t)

	mustDo(t, mux, http.MethodPost, "/api/state/filltubing/button", `{"hold_seconds":0.05}`)
	mustDo(t, mux, http.MethodPost, "/api/state/filltubing/button", `{"down":true}`)
	time.Sleep(200 * time.Millisecond)

	if !ps.FillTubingButtonDown() {
		t.Error("the earlier hold released a button pressed after it")
	}
	if got := fillTubingStreamed(h); got != 2 {
		t.Errorf("streamed %d FillTubingStateStreamResponses, want the two presses", got)
	}
}

func TestFillTubingButtonRefusals(t *testing.T) {
	_, ps, transport, mux := fillTubingHarness(t)

	for _, body := range []string{`{}`, `{"down":true,"hold_seconds":2}`, `{"hold_seconds":-1}`} {
		if code, _ := do(t, mux, http.MethodPost, "/api/state/filltubing/button", body); code != http.StatusBadRequest {
			t.Errorf("%s answered %d, want 400", body, code)
		}
	}

	ps.SetWorkflowMode(state.WorkflowNone)
	if code, _ := do(t, mux, http.MethodPost, "/api/state/filltubing/button", `{"down":true}`); code != http.StatusConflict {
		t.Errorf("pressing outside fill tubing mode answered %d, want 409", code)
	}
	if len(transport.notifications) != 0 {
		t.Errorf("refused presses notified %d fragment(s)", len(transport.notifications))
	}
}
