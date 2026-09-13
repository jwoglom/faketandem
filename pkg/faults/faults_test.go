package faults

import (
	"testing"
	"time"
)

func armOrFail(t *testing.T, r *Registry, f Fault) *Fault {
	t.Helper()
	armed, err := r.Arm(f)
	if err != nil {
		t.Fatalf("Arm(%+v): %v", f, err)
	}
	return armed
}

func TestArmDefaultsToTheNextMatchOnly(t *testing.T) {
	r := NewRegistry()
	armed := armOrFail(t, r, Fault{Kind: KindDropResponse, Message: "SetTempRateRequest"})

	if armed.Count != 1 {
		t.Errorf("count = %d, want an implicit 1", armed.Count)
	}

	target := Target{Message: "SetTempRateResponse", Characteristic: "Control"}
	if got := r.Match(target, KindDropResponse); got == nil {
		t.Fatal("the armed fault did not match the first time")
	}
	if got := r.Match(target, KindDropResponse); got != nil {
		t.Error("a next-one fault fired twice")
	}
	if len(r.List()) != 0 {
		t.Error("a spent fault stayed armed")
	}
}

func TestEveryFiresRepeatedlyAndCounts(t *testing.T) {
	r := NewRegistry()
	armOrFail(t, r, Fault{Kind: KindDropResponse, Every: true})

	target := Target{Message: "AnythingResponse"}
	for i := 0; i < 3; i++ {
		if r.Match(target, KindDropResponse) == nil {
			t.Fatalf("every-fault did not fire on match %d", i+1)
		}
	}

	listed := r.List()
	if len(listed) != 1 {
		t.Fatalf("every-fault should stay armed, got %d faults", len(listed))
	}
	if listed[0].Fired != 3 {
		t.Errorf("fired = %d, want 3", listed[0].Fired)
	}
}

func TestCountedFaultFiresExactlyNTimes(t *testing.T) {
	r := NewRegistry()
	armOrFail(t, r, Fault{Kind: KindDropResponse, Count: 2})

	target := Target{Message: "X"}
	for i := 0; i < 2; i++ {
		if r.Match(target, KindDropResponse) == nil {
			t.Fatalf("a count=2 fault did not fire on match %d", i+1)
		}
	}
	if r.Match(target, KindDropResponse) != nil {
		t.Error("a count=2 fault fired a third time")
	}
}

func TestScopingByMessageOpcodeAndCharacteristic(t *testing.T) {
	r := NewRegistry()
	armOrFail(t, r, Fault{Kind: KindDropResponse, Message: "SetTempRateRequest", Every: true})

	if r.Match(Target{Message: "InitiateBolusResponse"}, KindDropResponse) != nil {
		t.Error("a message-scoped fault matched an unrelated message")
	}
	// Either half of the request/response pair names the same exchange.
	if r.Match(Target{Message: "SetTempRateResponse"}, KindDropResponse) == nil {
		t.Error("scope by request name did not match the response")
	}
	if r.Match(Target{Message: "settemprateREQUEST"}, KindDropResponse) == nil {
		t.Error("message matching should be case-insensitive")
	}
	r.Clear()

	armOrFail(t, r, Fault{Kind: KindDropResponse, Opcode: 77, Every: true})
	if r.Match(Target{Opcode: 12, Message: "X"}, KindDropResponse) != nil {
		t.Error("an opcode-scoped fault matched a different opcode")
	}
	if r.Match(Target{Opcode: 77, Message: "X"}, KindDropResponse) == nil {
		t.Error("an opcode-scoped fault did not match its opcode")
	}
	r.Clear()

	armOrFail(t, r, Fault{Kind: KindDropResponse, Characteristic: "Control", Every: true})
	if r.Match(Target{Characteristic: "CurrentStatus"}, KindDropResponse) != nil {
		t.Error("a characteristic-scoped fault matched another characteristic")
	}
	if r.Match(Target{Characteristic: "control"}, KindDropResponse) == nil {
		t.Error("characteristic matching should be case-insensitive")
	}
}

func TestMatchFiltersByKind(t *testing.T) {
	r := NewRegistry()
	armOrFail(t, r, Fault{Kind: KindDelayResponse, DelayMs: 10, Every: true})

	if r.Match(Target{Message: "X"}, KindDropResponse) != nil {
		t.Error("asking for drop_response returned a delay_response fault")
	}
	if got := r.Match(Target{Message: "X"}, KindDropResponse, KindDelayResponse); got == nil {
		t.Fatal("the delay fault did not match when its kind was allowed")
	} else if got.Delay() != 10*time.Millisecond {
		t.Errorf("Delay() = %v, want 10ms", got.Delay())
	}
}

func TestPeekDoesNotConsume(t *testing.T) {
	r := NewRegistry()
	armOrFail(t, r, Fault{Kind: KindDropResponse})

	if r.Peek(Target{Message: "X"}, KindDropResponse) == nil {
		t.Fatal("Peek found nothing")
	}
	if r.Match(Target{Message: "X"}, KindDropResponse) == nil {
		t.Error("Peek consumed the fault")
	}
}

func TestValidationRejectsUnusableFaults(t *testing.T) {
	r := NewRegistry()

	for _, tc := range []struct {
		name  string
		fault Fault
	}{
		{"no kind", Fault{}},
		{"unknown kind", Fault{Kind: "explode"}},
		{"delay without a delay", Fault{Kind: KindDelayResponse}},
		{"bad disconnect timing", Fault{Kind: KindDisconnect, After: "someday"}},
		{"negative count", Fault{Kind: KindDropResponse, Count: -1}},
	} {
		if _, err := r.Arm(tc.fault); err == nil {
			t.Errorf("%s: Arm accepted a fault that could never fire", tc.name)
		}
	}

	// A disconnect with no timing defaults to cutting the link instead of
	// answering, which is the commoner case.
	armed := armOrFail(t, r, Fault{Kind: KindDisconnect})
	if armed.After != AfterRequest {
		t.Errorf("disconnect defaulted to after=%q, want %q", armed.After, AfterRequest)
	}
}

func TestRemoveAndClear(t *testing.T) {
	r := NewRegistry()
	first := armOrFail(t, r, Fault{Kind: KindDropResponse, Every: true})
	armOrFail(t, r, Fault{Kind: KindDelayResponse, DelayMs: 5, Every: true})

	if !r.Remove(first.ID) {
		t.Fatal("Remove reported the fault was not there")
	}
	if r.Remove(first.ID) {
		t.Error("Remove reported success twice for the same id")
	}
	if len(r.List()) != 1 {
		t.Fatalf("after Remove there are %d faults, want 1", len(r.List()))
	}
	if cleared := r.Clear(); cleared != 1 {
		t.Errorf("Clear reported %d, want 1", cleared)
	}
	if len(r.List()) != 0 {
		t.Error("faults survived Clear")
	}
}

func TestFirstArmedFaultWinsForTheSameTarget(t *testing.T) {
	r := NewRegistry()
	first := armOrFail(t, r, Fault{Kind: KindDropResponse, Every: true})
	armOrFail(t, r, Fault{Kind: KindDelayResponse, DelayMs: 5, Every: true})

	got := r.Match(Target{Message: "X"}, KindDropResponse, KindDelayResponse)
	if got == nil || got.ID != first.ID {
		t.Errorf("matched %+v, want the earlier-armed fault %d", got, first.ID)
	}
}
