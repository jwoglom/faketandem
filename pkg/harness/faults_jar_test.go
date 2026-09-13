package harness

import (
	"net/http"
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/protocol"
)

// TestErrorResponseFaultParsesAsErrorResponse reads the injected error back
// through the real cliparser jar, which is the only way to show the bytes are
// an ErrorResponse as pumpX2 defines one rather than merely a well-framed
// message with opcode 77 in it.
//
// Skipped unless FAKETANDEM_TEST_CLIPARSER_JAR is set.
func TestErrorResponseFaultParsesAsErrorResponse(t *testing.T) {
	bridge := testBridge(t)
	rig := newFaultRig(t)

	mustDo(t, rig.mux, http.MethodPost, "/api/faults",
		`{"kind":"error_response","message":"FaultProbeRequest","error_code":3}`)
	if err := rig.route(t); err != nil {
		t.Fatalf("RouteMessage: %v", err)
	}

	fragments := rig.client.CollectNotifications(1, time.Second)
	if len(fragments) != 1 {
		t.Fatalf("central received %d fragments, want the single-fragment ErrorResponse", len(fragments))
	}

	parsed, err := bridge.ParseMessage(bluetooth.CharCurrentStatus, fragments)
	if err != nil {
		t.Fatalf("cliparser could not parse the injected error: %v", err)
	}
	if parsed.MessageType != "ErrorResponse" {
		t.Fatalf("the injected error parses as %q, want ErrorResponse (cargo=%v)", parsed.MessageType, parsed.Cargo)
	}
	if parsed.Opcode != int(protocol.OpcodeErrorResponse) {
		t.Errorf("opcode = %d, want %d", parsed.Opcode, protocol.OpcodeErrorResponse)
	}
	if got := cargoInt(t, parsed, "requestCodeId"); got != probeOpcode {
		t.Errorf("requestCodeId = %d, want the rejected request's opcode %d", got, probeOpcode)
	}
	if got := cargoInt(t, parsed, "errorCodeId"); got != 3 {
		t.Errorf("errorCodeId = %d, want the armed error code 3", got)
	}
}
