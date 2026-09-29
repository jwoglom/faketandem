package handler

import (
	"fmt"
	"testing"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"
)

// These tests read the status a driver would decode: the request goes through the handler, and
// its response is parsed back through the real cliparser.
//
// Skipped unless FAKETANDEM_TEST_CLIPARSER_JAR is set.

func cartridgeModeStatus(t *testing.T, bridge *pumpx2.Bridge, ps *state.PumpState, request string) int {
	t.Helper()

	h := handlerFor(t, bridge, request)
	resp, err := h.HandleMessage(&pumpx2.ParsedMessage{TxID: 7, MessageType: request, Cargo: map[string]interface{}{}}, ps)
	if err != nil {
		t.Fatalf("%s handler failed: %v", request, err)
	}
	if resp == nil || resp.ResponseMessage == nil {
		t.Fatalf("%s produced no response", request)
	}

	parsed, err := bridge.ParseMessage(bluetooth.CharControl, resp.ResponseMessage.Packets)
	if err != nil {
		t.Fatalf("could not parse the %s answer back: %v", request, err)
	}
	var status int
	if _, err := fmt.Sscan(fmt.Sprint(parsed.Cargo["status"]), &status); err != nil {
		t.Fatalf("%s answer has no numeric status: %v", request, parsed.Cargo)
	}
	return status
}

func handlerFor(t *testing.T, bridge *pumpx2.Bridge, request string) MessageHandler {
	t.Helper()

	router := NewRouter(bridge, state.NewPumpState(), nil, nil, "go", "", "jar", "", "java", "")
	h, ok := router.handlers[request]
	if !ok {
		t.Fatalf("no handler registered for %s", request)
	}
	return h
}

func TestCartridgeModeRequestsAnswerWithTheModeTheyFind(t *testing.T) {
	bridge := testBridge(t)
	ps := state.NewPumpState()
	ps.SetPumpingSuspended(true)

	steps := []struct {
		request string
		status  int
		mode    state.WorkflowMode
	}{
		{"EnterChangeCartridgeModeRequest", 0, state.WorkflowChangeCartridge},
		{"EnterChangeCartridgeModeRequest", 1, state.WorkflowChangeCartridge},
		{"ExitChangeCartridgeModeRequest", 0, state.WorkflowNone},
		{"ExitChangeCartridgeModeRequest", 1, state.WorkflowNone},
		{"EnterFillTubingModeRequest", 0, state.WorkflowFillTubing},
		{"EnterChangeCartridgeModeRequest", 1, state.WorkflowFillTubing},
		{"ExitChangeCartridgeModeRequest", 1, state.WorkflowFillTubing},
		{"ExitFillTubingModeRequest", 0, state.WorkflowNone},
		{"ExitFillTubingModeRequest", 1, state.WorkflowNone},
		{"EnterChangeCartridgeModeRequest", 0, state.WorkflowChangeCartridge},
	}
	for i, step := range steps {
		if got := cartridgeModeStatus(t, bridge, ps, step.request); got != step.status {
			t.Errorf("step %d: %s answered status %d, want %d", i, step.request, got, step.status)
		}
		if got := ps.CurrentWorkflowMode(); got != step.mode {
			t.Errorf("step %d: after %s the pump is in %q, want %q", i, step.request, got, step.mode)
		}
	}
}

func TestEnterChangeCartridgeIsRefusedWhileDelivering(t *testing.T) {
	bridge := testBridge(t)
	ps := state.NewPumpState()

	if got := cartridgeModeStatus(t, bridge, ps, "EnterChangeCartridgeModeRequest"); got != 1 {
		t.Errorf("answered status %d while delivering, want 1", got)
	}
	if got := ps.CurrentWorkflowMode(); got != state.WorkflowNone {
		t.Errorf("a refused enter left the pump in %q", got)
	}
}
