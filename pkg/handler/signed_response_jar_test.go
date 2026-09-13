package handler

import (
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/bluetooth/virtualtest"
	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"
)

// These tests cover the signing of CONTROL responses end to end: a request
// goes through the router, the response cliparser encoded is re-signed with the
// pump's own key, and the bytes the central receives are checked against an
// independently written HMAC-SHA1 verifier.
//
// This is the one part of the wire format the cliparser subprocess cannot get
// right on its own. Its signing inputs arrive only through the environment, and
// it uses the raw ASCII bytes of PUMP_AUTHENTICATION_KEY as the HMAC key rather
// than decoding it, so a binary JPAKE-derived session key cannot reach it at
// all; left alone it signs with the ASCII of its own
// "IGNORE_HMAC_SIGNATURE_EXCEPTION" placeholder, which no driver will accept.
//
// Skipped unless FAKETANDEM_TEST_CLIPARSER_JAR is set.

// A binary key, deliberately: it is not valid ASCII and contains a NUL, so it
// could never have been passed to cliparser through the environment. Any test
// that passes here is proving the Go side did the signing.
var testAuthKey = []byte{
	0x00, 0xff, 0x10, 0x9a, 0x42, 0x7c, 0xde, 0x03,
	0x91, 0x55, 0xa0, 0x0b, 0xfe, 0x21, 0x38, 0xcc,
}

// signedRig is a pump, a router and a connected central over the virtual link.
type signedRig struct {
	router    *Router
	pumpState *state.PumpState
	client    *virtualtest.Client
}

func newSignedRig(t *testing.T) *signedRig {
	t.Helper()

	bridge := testBridge(t)

	transport, err := bluetooth.NewVirtual(bluetooth.VirtualOptions{Addr: "127.0.0.1:0", SerialNumber: "11223344"})
	if err != nil {
		t.Fatalf("NewVirtual: %v", err)
	}
	t.Cleanup(func() { _ = transport.Close() })
	if err := transport.SetPairingState(bluetooth.PairingStatePairStep1); err != nil {
		t.Fatalf("SetPairingState: %v", err)
	}

	pumpState := state.NewPumpState()
	pumpState.SetAuthenticated(testAuthKey)
	pumpState.UpdateTimeSinceReset()

	router := NewRouter(bridge, pumpState, transport, protocol.NewTransactionManager(time.Second),
		"go", "", "jar", "", "java", "")

	client := virtualtest.Dial(t, transport.Addr())
	client.Attach()
	client.Subscribe(bluetooth.ControlCharUUID, true)
	client.Subscribe(bluetooth.CurrentStatusCharUUID, true)

	return &signedRig{router: router, pumpState: pumpState, client: client}
}

// exchange routes one request on CONTROL and returns the fragments the central
// received.
func (r *signedRig) exchange(t *testing.T, messageType string, cargo map[string]interface{}) []string {
	t.Helper()

	err := r.router.RouteMessage(bluetooth.CharControl, &pumpx2.ParsedMessage{
		MessageType: messageType,
		TxID:        9,
		Cargo:       cargo,
	})
	if err != nil {
		t.Fatalf("routing %s: %v", messageType, err)
	}

	// Every signed CONTROL response here is two fragments; collect generously
	// and let the assertions below judge what arrived.
	got := r.client.CollectNotifications(4, 2*time.Second)
	if len(got) == 0 {
		t.Fatalf("the central received nothing in answer to %s", messageType)
	}
	return got
}

// TestSignedControlResponsesCarryAVerifiableHmac drives the control messages a
// driver uses to change therapy and checks each answer's trailer.
func TestSignedControlResponsesCarryAVerifiableHmac(t *testing.T) {
	cases := []struct {
		request  string
		response string
		cargo    map[string]interface{}
	}{
		{
			request:  "SuspendPumpingRequest",
			response: "SuspendPumpingResponse",
			cargo:    map[string]interface{}{},
		},
		{
			request:  "BolusPermissionRequest",
			response: "BolusPermissionResponse",
			cargo:    map[string]interface{}{},
		},
		{
			request:  "InitiateBolusRequest",
			response: "InitiateBolusResponse",
			cargo:    map[string]interface{}{"totalVolume": 2500, "bolusID": 42},
		},
		{
			request:  "SetTempRateRequest",
			response: "SetTempRateResponse",
			cargo:    map[string]interface{}{"percent": 150, "minutes": 30},
		},
	}

	for _, c := range cases {
		t.Run(c.response, func(t *testing.T) {
			if !protocol.IsSignedMessage(c.response) {
				t.Fatalf("%s is not in the signed-message table; this test is checking the wrong thing", c.response)
			}

			rig := newSignedRig(t)
			fragments := rig.exchange(t, c.request, c.cargo)

			trailer, ok, err := protocol.VerifySignedFragmentsHex(fragments, testAuthKey)
			if err != nil {
				t.Fatalf("%s did not come back as a well-formed signed message: %v", c.response, err)
			}
			if !ok {
				t.Errorf("%s carries an HMAC that does not validate against the pump's key", c.response)
			}
			if want := rig.pumpState.GetTimeSinceReset(); trailer.TimeSinceReset != want {
				t.Errorf("%s trailer timeSinceReset = %d, want the pump's %d",
					c.response, trailer.TimeSinceReset, want)
			}

			// The signature must be over the pump's key, not cliparser's
			// placeholder: verifying against the placeholder must fail.
			if _, wrongOK, err := protocol.VerifySignedFragmentsHex(
				fragments, []byte("IGNORE_HMAC_SIGNATURE_EXCEPTION")); err == nil && wrongOK {
				t.Errorf("%s is still signed with cliparser's placeholder key", c.response)
			}

			// The message must still be the one cliparser encoded: re-signing
			// replaces the trailer and the CRC, never the cargo.
			assertDecodesAs(t, fragments, c.response)
		})
	}
}

// assertDecodesAs parses the fragments back through cliparser, which is the
// check that the re-signed message is still a valid message of the expected
// type rather than merely well-signed bytes.
func assertDecodesAs(t *testing.T, fragments []string, want string) {
	t.Helper()

	bridge := testBridge(t)
	parsed, err := bridge.ParseMessage(bluetooth.CharControl, fragments)
	if err != nil {
		t.Fatalf("cliparser could not parse the re-signed message back: %v", err)
	}
	if parsed.MessageType != want {
		t.Errorf("the re-signed message parses back as %q, want %q", parsed.MessageType, want)
	}
}

// TestUnsignedResponsesAreUntouchedByResigning pins the other half of the
// contract: a response the catalog does not mark signed must reach the central
// exactly as cliparser encoded it, byte for byte.
func TestUnsignedResponsesAreUntouchedByResigning(t *testing.T) {
	rig := newSignedRig(t)

	for _, messageType := range []string{"ApiVersionRequest", "InsulinStatusRequest"} {
		t.Run(messageType, func(t *testing.T) {
			h, ok := rig.router.handlers[messageType]
			if !ok {
				t.Fatalf("no handler registered for %s", messageType)
			}
			resp, err := h.HandleMessage(
				&pumpx2.ParsedMessage{TxID: 9, MessageType: messageType, Cargo: map[string]interface{}{}},
				rig.pumpState)
			if err != nil {
				t.Fatalf("%s handler failed: %v", messageType, err)
			}
			if resp == nil || resp.ResponseMessage == nil {
				t.Fatalf("%s handler produced no response", messageType)
			}
			if protocol.IsSignedMessage(resp.ResponseMessage.MessageType) {
				t.Fatalf("%s answers with %s, which the table marks signed; this test needs an unsigned one",
					messageType, resp.ResponseMessage.MessageType)
			}

			want := strings.Join(resp.ResponseMessage.Packets, " ")
			got := strings.Join(rig.router.resign(resp.ResponseMessage).Packets, " ")
			if got != want {
				t.Errorf("re-signing changed an unsigned response:\n before: %s\n after:  %s", want, got)
			}
		})
	}
}

// TestResignReplacesOnlyTheTrailer takes one cliparser-encoded signed response
// apart before and after re-signing and shows the difference is confined to the
// 24-byte trailer and the CRC.
func TestResignReplacesOnlyTheTrailer(t *testing.T) {
	rig := newSignedRig(t)
	bridge := testBridge(t)

	encoded, err := bridge.EncodeMessage(9, "SuspendPumpingResponse", map[string]interface{}{"status": 0})
	if err != nil {
		t.Fatalf("cliparser encode failed: %v", err)
	}

	before := messageCargo(t, encoded.Packets)
	resigned := rig.router.resign(encoded)
	after := messageCargo(t, resigned.Packets)

	if hex.EncodeToString(before) != hex.EncodeToString(after) {
		t.Errorf("re-signing changed the cargo: %x -> %x", before, after)
	}
	if strings.Join(encoded.Packets, " ") == strings.Join(resigned.Packets, " ") {
		t.Error("re-signing left the message unchanged; the trailer was not rebuilt")
	}
	if _, ok, err := protocol.VerifySignedFragmentsHex(resigned.Packets, testAuthKey); err != nil || !ok {
		t.Errorf("the re-signed message does not verify: ok=%v err=%v", ok, err)
	}
}

// messageCargo returns a signed message's cargo: the payload with the 24-byte
// trailer taken off.
func messageCargo(t *testing.T, fragments []string) []byte {
	t.Helper()

	body, err := protocol.MessageBodyFromFragmentsHex(fragments)
	if err != nil {
		t.Fatalf("reassembling: %v", err)
	}
	parsed, err := protocol.ParseMessageBody(body)
	if err != nil {
		t.Fatalf("parsing the message body: %v", err)
	}
	cargo, err := parsed.Cargo(true)
	if err != nil {
		t.Fatalf("taking the trailer off: %v", err)
	}
	return cargo
}
