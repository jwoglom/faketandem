package harness

import (
	"net/http"
	"testing"

	"github.com/jwoglom/faketandem/pkg/state"
)

func TestSnapshotReportsTheWorkflowMode(t *testing.T) {
	_, ps, _, mux := testHarness(t)

	if got := mustDo(t, mux, http.MethodGet, "/api/state", "")["workflow_mode"]; got != "none" {
		t.Errorf("a fresh pump reports workflow_mode %v, want none", got)
	}

	ps.SetWorkflowMode(state.WorkflowFillTubing)
	if got := mustDo(t, mux, http.MethodGet, "/api/state", "")["workflow_mode"]; got != "fill_tubing" {
		t.Errorf("workflow_mode = %v, want fill_tubing", got)
	}
}

func TestStatePutStagesTheWorkflowMode(t *testing.T) {
	_, ps, transport, mux := testHarness(t)

	body := mustDo(t, mux, http.MethodPut, "/api/state", `{"workflow_mode":"change_cartridge"}`)
	if applied, _ := body["applied"].([]interface{}); len(applied) != 1 || applied[0] != "workflow_mode" {
		t.Errorf("applied = %v, want [workflow_mode]", body["applied"])
	}
	if got := ps.CurrentWorkflowMode(); got != state.WorkflowChangeCartridge {
		t.Errorf("pump is in %q after staging change_cartridge", got)
	}

	mustDo(t, mux, http.MethodPut, "/api/state", `{"workflow_mode":"none"}`)
	if got := ps.CurrentWorkflowMode(); got != state.WorkflowNone {
		t.Errorf("pump is in %q after staging none", got)
	}
	if bits := transport.qualifyingEventBits(); len(bits) != 0 {
		t.Errorf("staging a workflow mode raised qualifying events %v", bits)
	}
}

func TestStatePutRejectsAnUnknownWorkflowMode(t *testing.T) {
	_, ps, _, mux := testHarness(t)
	ps.SetWorkflowMode(state.WorkflowFillTubing)

	if code, _ := do(t, mux, http.MethodPut, "/api/state", `{"workflow_mode":"priming"}`); code != http.StatusBadRequest {
		t.Errorf("unknown workflow_mode answered %d, want 400", code)
	}
	if got := ps.CurrentWorkflowMode(); got != state.WorkflowFillTubing {
		t.Errorf("a rejected update moved the pump to %q", got)
	}
}
