package state

import "testing"

func suspendedPump(t *testing.T) *PumpState {
	t.Helper()
	ps := NewPumpState()
	ps.SetPumpingSuspended(true)
	return ps
}

func TestWorkflowStartsClosed(t *testing.T) {
	if got := NewPumpState().CurrentWorkflowMode(); got != WorkflowNone {
		t.Fatalf("a new pump is in %q, want %q", got, WorkflowNone)
	}
}

func TestEnterWorkflowOpensTheMode(t *testing.T) {
	for _, mode := range []WorkflowMode{WorkflowChangeCartridge, WorkflowFillTubing} {
		ps := suspendedPump(t)
		if !ps.EnterWorkflow(mode) {
			t.Fatalf("entering %s was refused on an idle, suspended pump", mode)
		}
		if got := ps.CurrentWorkflowMode(); got != mode {
			t.Errorf("after entering %s the pump is in %q", mode, got)
		}
	}
}

func TestEnterChangeCartridgeNeedsDeliverySuspended(t *testing.T) {
	ps := NewPumpState()
	if ps.EnterWorkflow(WorkflowChangeCartridge) {
		t.Fatal("change-cartridge was accepted while delivering")
	}
	if got := ps.CurrentWorkflowMode(); got != WorkflowNone {
		t.Errorf("a refused enter left the pump in %q", got)
	}
}

func TestEnterWorkflowIsRefusedWhileAnotherModeIsOpen(t *testing.T) {
	cases := []struct{ open, enter WorkflowMode }{
		{WorkflowChangeCartridge, WorkflowChangeCartridge},
		{WorkflowChangeCartridge, WorkflowFillTubing},
		{WorkflowFillTubing, WorkflowFillTubing},
		{WorkflowFillTubing, WorkflowChangeCartridge},
	}
	for _, c := range cases {
		t.Run(string(c.open)+"_then_"+string(c.enter), func(t *testing.T) {
			ps := suspendedPump(t)
			ps.SetWorkflowMode(c.open)
			if ps.EnterWorkflow(c.enter) {
				t.Errorf("entering %s was accepted with %s open", c.enter, c.open)
			}
			if got := ps.CurrentWorkflowMode(); got != c.open {
				t.Errorf("a refused enter moved the pump from %s to %s", c.open, got)
			}
		})
	}
}

func TestExitWorkflowOnlyLeavesTheModeThePumpIsIn(t *testing.T) {
	ps := suspendedPump(t)
	if ps.ExitWorkflow(WorkflowChangeCartridge) {
		t.Error("exited change-cartridge on an idle pump")
	}

	ps.SetWorkflowMode(WorkflowFillTubing)
	if ps.ExitWorkflow(WorkflowChangeCartridge) {
		t.Error("exited change-cartridge while in fill-tubing")
	}
	if got := ps.CurrentWorkflowMode(); got != WorkflowFillTubing {
		t.Errorf("a refused exit left the pump in %q, want it still in fill-tubing", got)
	}

	if !ps.ExitWorkflow(WorkflowFillTubing) {
		t.Fatal("exit of the open mode was refused")
	}
	if got := ps.CurrentWorkflowMode(); got != WorkflowNone {
		t.Errorf("after exiting the pump is in %q", got)
	}
}

// The field incident: fill-tubing was left open, and every change-cartridge attempt after it was
// refused, along with the exit that a driver sends to back out.
func TestAbandonedFillTubingBlocksChangeCartridgeUntilItIsExited(t *testing.T) {
	ps := suspendedPump(t)
	if !ps.EnterWorkflow(WorkflowFillTubing) {
		t.Fatal("fill-tubing was refused")
	}

	if ps.EnterWorkflow(WorkflowChangeCartridge) {
		t.Fatal("change-cartridge was accepted with fill-tubing open")
	}
	if ps.ExitWorkflow(WorkflowChangeCartridge) {
		t.Fatal("exit of change-cartridge succeeded although the pump was never in it")
	}

	if !ps.ExitWorkflow(WorkflowFillTubing) {
		t.Fatal("exit of fill-tubing was refused")
	}
	if !ps.EnterWorkflow(WorkflowChangeCartridge) {
		t.Fatal("change-cartridge was still refused after fill-tubing was exited")
	}
}

func TestParseWorkflowMode(t *testing.T) {
	for _, name := range []string{"none", "change_cartridge", "fill_tubing"} {
		if mode, ok := ParseWorkflowMode(name); !ok || string(mode) != name {
			t.Errorf("ParseWorkflowMode(%q) = %q, %v", name, mode, ok)
		}
	}
	if _, ok := ParseWorkflowMode("priming"); ok {
		t.Error("an unknown mode name parsed")
	}
}
