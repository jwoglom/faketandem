package harness

import (
	"net/http"
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/bluetooth/virtualtest"
	"github.com/jwoglom/faketandem/pkg/faults"
	"github.com/jwoglom/faketandem/pkg/handler"
	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/reqlog"
	"github.com/jwoglom/faketandem/pkg/state"
)

// These tests drive faults over a real loopback virtual-GATT link, so what is
// asserted is what a central actually receives -- including the fragments it
// does NOT receive, which is the whole point of the fault injector.
//
// They deliberately avoid cliparser: a probe handler returns pre-built
// fragments, so the tests exercise the fault and transport paths without a JVM
// anywhere near them.

const (
	probeRequest  = "FaultProbeRequest"
	probeResponse = "FaultProbeResponse"
	probeOpcode   = 4242
)

// probeHandler answers the probe request with a three-fragment response and a
// state change, so a test can tell "the pump acted" apart from "the pump
// answered".
type probeHandler struct {
	fragments []string
}

func (h *probeHandler) MessageType() string { return probeRequest }
func (h *probeHandler) RequiresAuth() bool  { return false }

func (h *probeHandler) HandleMessage(msg *pumpx2.ParsedMessage, _ *state.PumpState) (*handler.Response, error) {
	return &handler.Response{
		ResponseMessage: &pumpx2.EncodedMessage{
			MessageType: probeResponse,
			TxID:        msg.TxID,
			Opcode:      probeOpcode,
			Packets:     h.fragments,
		},
		Immediate: true,
		StateChanges: []handler.StateChange{
			{Type: handler.StateChangeSuspend, Data: true},
		},
	}, nil
}

// faultRig is a pump, a router, a virtual link and a connected central.
type faultRig struct {
	transport *bluetooth.VirtualTransport
	router    *handler.Router
	pumpState *state.PumpState
	requests  *reqlog.Log
	registry  *faults.Registry
	client    *virtualtest.Client
	mux       http.Handler
	fragments []string
}

func newFaultRig(t *testing.T) *faultRig {
	t.Helper()

	transport, err := bluetooth.NewVirtual(bluetooth.VirtualOptions{Addr: "127.0.0.1:0", SerialNumber: "11223344"})
	if err != nil {
		t.Fatalf("NewVirtual: %v", err)
	}
	t.Cleanup(func() { _ = transport.Close() })
	if err := transport.SetPairingState(bluetooth.PairingStatePairStep1); err != nil {
		t.Fatalf("SetPairingState: %v", err)
	}

	ps := state.NewPumpState()
	ps.SetClock(state.NewFrozenClock(testInstant))

	router := handler.NewRouter(nil, ps, transport, protocol.NewTransactionManager(time.Second),
		"go", "", "jar", "", "java", "")

	fragments := []string{
		virtualtest.HexPacket(0xa1, 18),
		virtualtest.HexPacket(0xa2, 18),
		virtualtest.HexPacket(0xa3, 4),
	}
	router.RegisterHandler(&probeHandler{fragments: fragments})

	log := reqlog.New(64)
	log.SetClock(ps.Now)
	registry := faults.NewRegistry()
	router.SetRequestLog(log)
	router.SetFaultRegistry(registry)

	h := New(Options{
		PumpState: ps,
		Simulator: state.NewSimulator(ps, time.Second),
		Transport: transport,
		Router:    router,
		Requests:  log,
		Faults:    registry,
	})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	client := virtualtest.Dial(t, transport.Addr())
	client.Attach()
	client.Subscribe(bluetooth.CurrentStatusCharUUID, true)

	return &faultRig{
		transport: transport,
		router:    router,
		pumpState: ps,
		requests:  log,
		registry:  registry,
		client:    client,
		mux:       mux,
		fragments: fragments,
	}
}

// route pushes one probe request through the router, as the write handler
// would after reassembly and parsing.
func (r *faultRig) route(t *testing.T) error {
	t.Helper()

	return r.router.RouteMessage(bluetooth.CharCurrentStatus, &pumpx2.ParsedMessage{
		MessageType: probeRequest,
		TxID:        7,
		Opcode:      probeOpcode,
		Cargo:       map[string]interface{}{"probe": 1},
	})
}

// lastResponseEntry returns the most recent response record in the log.
func (r *faultRig) lastResponseEntry(t *testing.T) reqlog.Entry {
	t.Helper()

	entries := r.requests.All()
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Kind == reqlog.KindResponse {
			return entries[i]
		}
	}
	t.Fatal("no response entry in the request log")
	return reqlog.Entry{}
}

func TestVirtualLinkDeliversTheWholeResponseWithNoFaults(t *testing.T) {
	rig := newFaultRig(t)

	if err := rig.route(t); err != nil {
		t.Fatalf("RouteMessage: %v", err)
	}

	got := rig.client.CollectNotifications(3, time.Second)
	if len(got) != 3 {
		t.Fatalf("central received %d fragments, want 3: %v", len(got), got)
	}
	for i, frag := range got {
		if frag != rig.fragments[i] {
			t.Errorf("fragment %d = %s, want %s", i, frag, rig.fragments[i])
		}
	}

	// The log records both halves of the exchange.
	entries := rig.requests.All()
	if len(entries) < 2 {
		t.Fatalf("log has %d entries, want the request and the response", len(entries))
	}
	if entries[0].Kind != reqlog.KindRequest || entries[0].Message != probeRequest {
		t.Errorf("first log entry = %+v", entries[0])
	}
	if entries[0].Cargo["probe"] != 1 {
		t.Errorf("request cargo was not logged: %v", entries[0].Cargo)
	}
	resp := rig.lastResponseEntry(t)
	if resp.FragmentsSent != 3 || resp.Fault != "" {
		t.Errorf("response entry = %+v, want 3 fragments and no fault", resp)
	}
}

func TestDropResponseAppliesStateButSendsNothing(t *testing.T) {
	rig := newFaultRig(t)

	mustDo(t, rig.mux, http.MethodPost, "/api/faults",
		`{"kind":"drop_response","message":"FaultProbeRequest"}`)

	if err := rig.route(t); err != nil {
		t.Fatalf("RouteMessage: %v", err)
	}

	if got := rig.client.CollectNotifications(1, 250*time.Millisecond); len(got) != 0 {
		t.Errorf("central received %d fragments, want none", len(got))
	}

	// This is the case the whole fault exists for: the pump acted on the
	// request, and only the answer was lost.
	if !rig.pumpState.IsPumpingSuspended() {
		t.Error("the handler's state change was not applied; a dropped response must not undo the request")
	}

	resp := rig.lastResponseEntry(t)
	if resp.Fault != faults.KindDropResponse || resp.FragmentsSent != 0 {
		t.Errorf("response entry = %+v, want a drop_response with 0 fragments sent", resp)
	}
	if len(resp.Fragments) != 3 {
		t.Errorf("the log should still hold the %d fragments that were never sent", len(resp.Fragments))
	}

	// The fault was armed for the next message only, so the following one is
	// answered normally.
	if err := rig.route(t); err != nil {
		t.Fatalf("second RouteMessage: %v", err)
	}
	if got := rig.client.CollectNotifications(3, time.Second); len(got) != 3 {
		t.Errorf("the next response was also dropped: %d fragments", len(got))
	}
}

func TestDelayResponseHoldsTheAnswerBack(t *testing.T) {
	rig := newFaultRig(t)

	mustDo(t, rig.mux, http.MethodPost, "/api/faults",
		`{"kind":"delay_response","message":"FaultProbeRequest","delay_ms":150}`)

	start := time.Now()
	if err := rig.route(t); err != nil {
		t.Fatalf("RouteMessage: %v", err)
	}
	elapsed := time.Since(start)

	if got := rig.client.CollectNotifications(3, time.Second); len(got) != 3 {
		t.Fatalf("central received %d fragments, want all 3 after the delay", len(got))
	}
	if elapsed < 150*time.Millisecond {
		t.Errorf("the response took %v, want at least the 150ms delay", elapsed)
	}

	resp := rig.lastResponseEntry(t)
	if resp.Fault != faults.KindDelayResponse || resp.FragmentsSent != 3 {
		t.Errorf("response entry = %+v", resp)
	}
}

func TestErrorResponseUsesTheEncoderHookAndDegradesWithoutIt(t *testing.T) {
	rig := newFaultRig(t)

	// With no encoder installed the fault must not pretend to have worked.
	mustDo(t, rig.mux, http.MethodPost, "/api/faults",
		`{"kind":"error_response","message":"FaultProbeRequest","error_code":3}`)
	if err := rig.route(t); err != nil {
		t.Fatalf("RouteMessage: %v", err)
	}
	if got := rig.client.CollectNotifications(1, 250*time.Millisecond); len(got) != 0 {
		t.Errorf("central received %d fragments with no encoder installed", len(got))
	}
	resp := rig.lastResponseEntry(t)
	if resp.Fault != faults.KindErrorResponse || resp.FragmentsSent != 0 {
		t.Errorf("response entry = %+v", resp)
	}
	if resp.Note == "" {
		t.Error("the log does not say why nothing was sent")
	}

	// With an encoder installed -- as the native packet encoder will provide
	// -- the error goes out in place of the real response.
	errorFragment := virtualtest.HexPacket(0xee, 6)
	var gotTxID, gotCode, gotOpcode int
	var gotName string
	handler.ErrorResponseEncoder = func(txID, errorCode, requestOpcode int, requestName string) (*pumpx2.EncodedMessage, error) {
		gotTxID, gotCode, gotOpcode, gotName = txID, errorCode, requestOpcode, requestName
		return &pumpx2.EncodedMessage{
			MessageType: "ErrorResponse",
			TxID:        txID,
			Opcode:      77,
			Packets:     []string{errorFragment},
		}, nil
	}
	t.Cleanup(func() { handler.ErrorResponseEncoder = nil })

	mustDo(t, rig.mux, http.MethodPost, "/api/faults",
		`{"kind":"error_response","message":"FaultProbeRequest","error_code":3}`)
	if err := rig.route(t); err != nil {
		t.Fatalf("RouteMessage: %v", err)
	}

	got := rig.client.CollectNotifications(1, time.Second)
	if len(got) != 1 || got[0] != errorFragment {
		t.Fatalf("central received %v, want the single error fragment %s", got, errorFragment)
	}
	if gotTxID != 7 || gotCode != 3 || gotOpcode != probeOpcode || gotName != probeResponse {
		t.Errorf("encoder called with txID=%d code=%d opcode=%d name=%q",
			gotTxID, gotCode, gotOpcode, gotName)
	}
	if resp := rig.lastResponseEntry(t); resp.Message != "ErrorResponse" {
		t.Errorf("the log recorded %q, want the ErrorResponse that actually went out", resp.Message)
	}
}

func TestDisconnectAfterRequestCutsTheLinkWithNoAnswer(t *testing.T) {
	rig := newFaultRig(t)

	mustDo(t, rig.mux, http.MethodPost, "/api/faults",
		`{"kind":"disconnect","message":"FaultProbeRequest","after":"request"}`)

	if err := rig.route(t); err != nil {
		t.Fatalf("RouteMessage: %v", err)
	}

	rig.client.ExpectDisconnect(time.Second)
	if !rig.pumpState.IsPumpingSuspended() {
		t.Error("the state change was not applied before the link was cut")
	}
	if resp := rig.lastResponseEntry(t); resp.FragmentsSent != 0 || resp.Fault != faults.KindDisconnect {
		t.Errorf("response entry = %+v", resp)
	}
	waitFor(t, "the transport to report no central", func() bool { return !rig.transport.IsConnected() })
}

func TestDisconnectMidResponseSendsExactlyKFragments(t *testing.T) {
	rig := newFaultRig(t)

	mustDo(t, rig.mux, http.MethodPost, "/api/faults",
		`{"kind":"disconnect","message":"FaultProbeRequest","after":"partial_response","fragments_sent":2}`)

	if err := rig.route(t); err != nil {
		t.Fatalf("RouteMessage: %v", err)
	}

	got := rig.client.CollectNotifications(3, 500*time.Millisecond)
	if len(got) != 2 {
		t.Fatalf("central received %d fragments, want exactly the 2 configured", len(got))
	}
	if got[0] != rig.fragments[0] || got[1] != rig.fragments[1] {
		t.Errorf("the wrong fragments arrived: %v", got)
	}

	resp := rig.lastResponseEntry(t)
	if resp.FragmentsSent != 2 || len(resp.Fragments) != 3 {
		t.Errorf("response entry = %+v, want 2 of 3 fragments sent", resp)
	}
	waitFor(t, "the transport to report no central", func() bool { return !rig.transport.IsConnected() })
}

func TestRadioOffRefusesConnectionsAndRadioOnRestoresThem(t *testing.T) {
	rig := newFaultRig(t)

	mustDo(t, rig.mux, http.MethodPost, "/api/faults", `{"kind":"radio_off"}`)

	rig.client.ExpectDisconnect(time.Second)
	waitFor(t, "the transport to drop the central", func() bool { return !rig.transport.IsConnected() })
	if rig.transport.RadioEnabled() {
		t.Error("RadioEnabled() is still true after radio_off")
	}

	// A central that tries again while the radio is off is turned away.
	refused := virtualtest.Dial(t, rig.transport.Addr())
	msg := refused.Recv()
	if msg.Type != "disconnect" || msg.Reason != "radio_off" {
		t.Errorf("reconnect while the radio is off got %+v, want a radio_off disconnect", msg)
	}

	body := mustDo(t, rig.mux, http.MethodPost, "/api/faults", `{"kind":"radio_on"}`)
	if body["radio_enabled"] != true {
		t.Fatalf("radio_on reported %v", body["radio_enabled"])
	}

	reconnected := virtualtest.Dial(t, rig.transport.Addr())
	reconnected.Attach()
	reconnected.Subscribe(bluetooth.CurrentStatusCharUUID, true)
	if err := rig.route(t); err != nil {
		t.Fatalf("RouteMessage after the radio came back: %v", err)
	}
	if got := reconnected.CollectNotifications(3, time.Second); len(got) != 3 {
		t.Errorf("the reconnected central received %d fragments, want 3", len(got))
	}
}

func TestNotifyFilterDropsIndividualFragments(t *testing.T) {
	rig := newFaultRig(t)

	// The transport-level seam works in single fragments, which is what
	// fragment-granular faults need and what the router (which works in whole
	// messages) cannot express.
	rig.transport.SetNotifyFilter(func(_ bluetooth.CharacteristicType, data []byte) bool {
		return len(data) != 18
	})

	if err := rig.route(t); err != nil {
		t.Fatalf("RouteMessage: %v", err)
	}

	got := rig.client.CollectNotifications(3, 500*time.Millisecond)
	if len(got) != 1 || got[0] != rig.fragments[2] {
		t.Errorf("central received %v, want only the short third fragment", got)
	}
}

// waitFor polls cond until it holds or a second elapses.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}
