package handler

import (
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/controlstream"
	"github.com/jwoglom/faketandem/pkg/state"
)

func fillTubingRouter(t *testing.T) (*Router, *state.PumpState, *recordingTransport) {
	t.Helper()
	ps := state.NewPumpState()
	ps.SetPumpingSuspended(true)
	ps.Lock()
	ps.AuthKey = []byte("0123456789abcdef0123456789abcdef")
	ps.Unlock()
	transport := &recordingTransport{}
	return NewRouter(testBridge(t), ps, transport, nil, "go", "", "jar", "", "java", ""), ps, transport
}

// streamedButtonStates reassembles every CONTROL_STREAM message the router sent and checks each is a
// FillTubingStateStreamResponse under txID, signed with the session key.
func streamedButtonStates(t *testing.T, ps *state.PumpState, transport *recordingTransport, txID uint8) []int32 {
	t.Helper()
	var states []int32
	var body []byte
	for _, n := range transport.notifications {
		if n.charType != bluetooth.CharControlStream {
			continue
		}
		body = append(body, n.data[2:]...)
		if n.data[0] != 0 {
			continue
		}

		if _, ok, err := protocol.VerifySignedMessage(body, ps.GetAuthKey()); err != nil || !ok {
			t.Fatalf("stream message %x is not signed with the session key (err %v)", body, err)
		}
		parsed, err := protocol.ParseMessageBody(body)
		if err != nil {
			t.Fatal(err)
		}
		if parsed.Opcode != protocol.OpcodeFillTubingStateStreamResponse || parsed.TxID != txID {
			t.Fatalf("stream message opcode %d txID %d, want FillTubingStateStreamResponse under %d", parsed.Opcode, parsed.TxID, txID)
		}
		response, err := controlstream.NewFillTubingStateStreamResponseFromCargo(parsed.Payload)
		if err != nil {
			t.Fatalf("FillTubingStateStreamResponse does not parse: %v", err)
		}
		states = append(states, response.ButtonState)
		body = nil
	}
	return states
}

func TestFillTubingButtonStreamsUnderTheEnterRequestsTxID(t *testing.T) {
	router, ps, transport := fillTubingRouter(t)

	if _, err := router.handlers["EnterFillTubingModeRequest"].HandleMessage(
		&pumpx2.ParsedMessage{TxID: 0x76, MessageType: "EnterFillTubingModeRequest", Cargo: map[string]interface{}{}}, ps); err != nil {
		t.Fatal(err)
	}
	for _, down := range []bool{true, false, true, false} {
		if err := router.SetFillTubingButton(down); err != nil {
			t.Fatalf("SetFillTubingButton(%v): %v", down, err)
		}
		if ps.FillTubingButtonDown() != down {
			t.Errorf("after SetFillTubingButton(%v) the pump reads the button down=%v", down, !down)
		}
	}

	got := streamedButtonStates(t, ps, transport, 0x76)
	if want := []int32{1, 0, 1, 0}; !reflect.DeepEqual(got, want) {
		t.Errorf("streamed button states %v, want %v", got, want)
	}
}

func TestFillTubingButtonIsRefusedOutsideFillTubingMode(t *testing.T) {
	router, ps, transport := fillTubingRouter(t)

	if err := router.SetFillTubingButton(true); err == nil {
		t.Error("the button streamed with no fill tubing mode open")
	}
	ps.SetWorkflowMode(state.WorkflowChangeCartridge)
	if err := router.SetFillTubingButton(true); err == nil {
		t.Error("the button streamed in change cartridge mode")
	}
	if len(transport.notifications) != 0 {
		t.Errorf("a refused press notified %d fragment(s)", len(transport.notifications))
	}
}

func TestFillTubingButtonIsRefusedBeforeAuthentication(t *testing.T) {
	router, ps, transport := fillTubingRouter(t)
	ps.Lock()
	ps.AuthKey = nil
	ps.Unlock()
	ps.SetWorkflowMode(state.WorkflowFillTubing)

	if err := router.SetFillTubingButton(true); err == nil {
		t.Error("the button streamed with no session key to sign it")
	}
	if ps.FillTubingButtonDown() || len(transport.notifications) != 0 {
		t.Error("a refused press changed the button or notified")
	}
}

func TestExitingFillTubingReleasesTheButton(t *testing.T) {
	router, ps, _ := fillTubingRouter(t)
	if !ps.EnterFillTubing(3) {
		t.Fatal("fill tubing mode refused")
	}
	if err := router.SetFillTubingButton(true); err != nil {
		t.Fatal(err)
	}
	if !ps.ExitWorkflow(state.WorkflowFillTubing) {
		t.Fatal("exit refused")
	}
	if ps.FillTubingButtonDown() {
		t.Error("the button still reads down after fill tubing mode closed")
	}
}
