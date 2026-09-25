package harness

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
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
	// A byte-sized opcode, because the wire format has exactly one byte for
	// it: an ErrorResponse naming the rejected request has to be able to
	// carry this value.
	probeOpcode = 61
	// probeTxID is the transaction every probe exchange uses.
	probeTxID = 7
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

	fragments := probeFragments()
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

// probeFragments builds the probe response's three BLE notifications with real
// [remaining][txId] framing -- 18, 18 and 4 bytes, the shape an 18-byte-chunked
// response actually has.
//
// The framing matters to fragment-level faults: the remaining-fragment counter
// is the only thing below the router that says where one message ends and the
// next begins, so a probe made of unframed filler would make "the second
// fragment of every message" untestable.
func probeFragments() []string {
	payloadSizes := []int{16, 16, 2}
	markers := []byte{0xa1, 0xa2, 0xa3}

	fragments := make([]string, 0, len(markers))
	for i, marker := range markers {
		fragment := []byte{byte(len(markers) - i - 1), probeTxID}
		for j := 0; j < payloadSizes[i]; j++ {
			fragment = append(fragment, marker)
		}
		fragments = append(fragments, hex.EncodeToString(fragment))
	}
	return fragments
}

// route pushes one probe request through the router, as the write handler
// would after reassembly and parsing.
func (r *faultRig) route(t *testing.T) error {
	t.Helper()

	return r.router.RouteMessage(bluetooth.CharCurrentStatus, &pumpx2.ParsedMessage{
		MessageType: probeRequest,
		TxID:        probeTxID,
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

func TestErrorResponseFaultSendsARealErrorResponse(t *testing.T) {
	rig := newFaultRig(t)

	mustDo(t, rig.mux, http.MethodPost, "/api/faults",
		`{"kind":"error_response","message":"FaultProbeRequest","error_code":3}`)
	if err := rig.route(t); err != nil {
		t.Fatalf("RouteMessage: %v", err)
	}

	got := rig.client.CollectNotifications(1, time.Second)
	if len(got) != 1 {
		t.Fatalf("central received %d fragments, want the single-fragment ErrorResponse", len(got))
	}

	// Read it the way the driver does: off the wire, through the reassembler.
	body := reassemble(t, bluetooth.CharCurrentStatus, got)
	parsed, err := protocol.ParseMessageBody(body)
	if err != nil {
		t.Fatalf("the ErrorResponse is not a well-formed message: %v", err)
	}
	if parsed.Opcode != protocol.OpcodeErrorResponse {
		t.Errorf("opcode = %d, want %d (ErrorResponse)", parsed.Opcode, protocol.OpcodeErrorResponse)
	}
	if parsed.TxID != 7 {
		t.Errorf("txId = %d, want the 7 of the request it answers", parsed.TxID)
	}
	if len(parsed.Payload) != 2 {
		t.Fatalf("cargo = %x, want the two bytes [requestCodeId][errorCodeId]", parsed.Payload)
	}
	if int(parsed.Payload[0]) != probeOpcode {
		t.Errorf("requestCodeId = %d, want the rejected request's opcode %d", parsed.Payload[0], probeOpcode)
	}
	if parsed.Payload[1] != 3 {
		t.Errorf("errorCodeId = %d, want the armed error code 3", parsed.Payload[1])
	}

	// The state change still happened: an error_response fault answers with an
	// error, it does not undo the request.
	if !rig.pumpState.IsPumpingSuspended() {
		t.Error("the state change was not applied before the error went out")
	}
	if resp := rig.lastResponseEntry(t); resp.Message != "ErrorResponse" || resp.FragmentsSent != 1 {
		t.Errorf("the log recorded %+v, want the ErrorResponse that actually went out", resp)
	}
}

func TestErrorResponseFaultDegradesWithNoEncoder(t *testing.T) {
	rig := newFaultRig(t)

	// Clearing the hook must not make the fault pretend to have worked.
	previous := handler.ErrorResponseEncoder
	handler.ErrorResponseEncoder = nil
	t.Cleanup(func() { handler.ErrorResponseEncoder = previous })

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
}

// reassemble feeds hex fragments through the real reassembler and returns the
// message body, so a test asserts on what a central reconstructs rather than on
// the fragments the emulator happened to emit.
func reassemble(t *testing.T, charType bluetooth.CharacteristicType, fragmentsHex []string) []byte {
	t.Helper()

	r := protocol.NewReassembler(time.Second)
	t.Cleanup(r.Stop)

	for i, fragmentHex := range fragmentsHex {
		fragment, err := hex.DecodeString(fragmentHex)
		if err != nil {
			t.Fatalf("fragment %d is not hex: %v", i, err)
		}
		body, _, complete, err := r.AddPacket(charType, fragment)
		if err != nil {
			t.Fatalf("reassembling fragment %d: %v", i, err)
		}
		if complete {
			if i != len(fragmentsHex)-1 {
				t.Fatalf("the message completed at fragment %d of %d", i+1, len(fragmentsHex))
			}
			return body
		}
	}
	t.Fatalf("the %d fragments never completed a message", len(fragmentsHex))
	return nil
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

func TestDropFragmentByIndexRemovesTheSamePositionOfEveryMessage(t *testing.T) {
	rig := newFaultRig(t)

	mustDo(t, rig.mux, http.MethodPost, "/api/faults",
		`{"kind":"drop_fragment","characteristic":"CurrentStatus","index":1,"every":true}`)

	// Two messages, so the position is shown to reset at each message
	// boundary rather than running on across the stream.
	for i := 0; i < 2; i++ {
		if err := rig.route(t); err != nil {
			t.Fatalf("RouteMessage %d: %v", i, err)
		}
	}

	got := rig.client.CollectNotifications(6, time.Second)
	want := []string{rig.fragments[0], rig.fragments[2], rig.fragments[0], rig.fragments[2]}
	if len(got) != len(want) {
		t.Fatalf("central received %d fragments, want %d (the middle one of each message dropped)", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("fragment %d = %s, want %s", i, got[i], want[i])
		}
	}

	// The pump still believes it sent everything: fragment loss happens below
	// the router, so the request log records a fully sent response.
	if resp := rig.lastResponseEntry(t); resp.FragmentsSent != 3 || resp.Fault != "" {
		t.Errorf("response entry = %+v, want the pump's own record of a complete send", resp)
	}
}

func TestDropFragmentEveryNthThinsTheStream(t *testing.T) {
	rig := newFaultRig(t)

	mustDo(t, rig.mux, http.MethodPost, "/api/faults",
		`{"kind":"drop_fragment","every_nth":2,"every":true}`)

	for i := 0; i < 2; i++ {
		if err := rig.route(t); err != nil {
			t.Fatalf("RouteMessage %d: %v", i, err)
		}
	}

	// Six fragments go out; every second one is lost, regardless of where the
	// message boundaries fall.
	got := rig.client.CollectNotifications(6, time.Second)
	want := []string{rig.fragments[0], rig.fragments[2], rig.fragments[1]}
	if len(got) != len(want) {
		t.Fatalf("central received %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("fragment %d = %s, want %s", i, got[i], want[i])
		}
	}
}

func TestDropFragmentCountExpires(t *testing.T) {
	rig := newFaultRig(t)

	// No count and no every: the default "next one only".
	mustDo(t, rig.mux, http.MethodPost, "/api/faults", `{"kind":"drop_fragment","index":0}`)

	for i := 0; i < 2; i++ {
		if err := rig.route(t); err != nil {
			t.Fatalf("RouteMessage %d: %v", i, err)
		}
	}

	got := rig.client.CollectNotifications(6, time.Second)
	want := []string{
		rig.fragments[1], rig.fragments[2],
		rig.fragments[0], rig.fragments[1], rig.fragments[2],
	}
	if len(got) != len(want) {
		t.Fatalf("central received %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("fragment %d = %s, want %s", i, got[i], want[i])
		}
	}

	body := mustDo(t, rig.mux, http.MethodGet, "/api/faults", "")
	if armed, ok := body["faults"].([]interface{}); !ok || len(armed) != 0 {
		t.Errorf("the spent fault is still armed: %v", body["faults"])
	}
}

func TestDropFragmentRejectsScopingItCannotHonor(t *testing.T) {
	rig := newFaultRig(t)

	// A fragment carries no message identity, so a message- or opcode-scoped
	// drop_fragment must be refused rather than quietly applied to everything.
	for _, body := range []string{
		`{"kind":"drop_fragment","index":0,"message":"FaultProbeRequest"}`,
		`{"kind":"drop_fragment","index":0,"opcode":61}`,
		`{"kind":"drop_fragment"}`,
		`{"kind":"drop_fragment","index":0,"every_nth":2}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/faults", strings.NewReader(body))
		rec := httptest.NewRecorder()
		rig.mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("arming %s returned %d, want 400", body, rec.Code)
		}
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
