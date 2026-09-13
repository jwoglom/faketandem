package harness

import (
	"encoding/hex"
	"net/http"
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/protocol"
)

// These tests drive the ATT MTU over a real loopback virtual-GATT link, so
// what is asserted is the framing a central actually receives.
//
// The emulator used to fragment every response at the 23-byte ATT MTU floor --
// 20-byte notifications, 18 bytes of message each -- faithfully to what
// pumpX2's Packetize and the cliparser jar produce. A real Mobi paired with the
// official iOS app negotiates a much larger MTU, so most responses arrive in a
// single notification. A driver's reassembler has to handle both, and until now
// only the fragmented half could be exercised here.

// TestDefaultMTUKeepsTheFragmentedFraming is the guard on the default: nothing
// about the wire bytes moves unless a test asks for a different MTU.
func TestDefaultMTUKeepsTheFragmentedFraming(t *testing.T) {
	rig := newFaultRig(t)

	snapshot := mustDo(t, rig.mux, http.MethodGet, "/api/transport", "")
	if got := snapshot["att_mtu"]; got != float64(bluetooth.DefaultATTMTU) {
		t.Errorf("att_mtu = %v, want the %d default", got, bluetooth.DefaultATTMTU)
	}
	if got := snapshot["mtu_configurable"]; got != true {
		t.Errorf("mtu_configurable = %v, want true for the virtual transport", got)
	}

	if err := rig.route(t); err != nil {
		t.Fatalf("RouteMessage: %v", err)
	}

	got := rig.client.CollectNotifications(3, time.Second)
	if len(got) != 3 {
		t.Fatalf("central received %d fragments, want the 3 of the default framing: %v", len(got), got)
	}
	for i, frag := range got {
		if frag != rig.fragments[i] {
			t.Errorf("fragment %d = %s, want %s (byte-identical to what the encoder produced)", i, frag, rig.fragments[i])
		}
	}
}

// TestLargeMTUDeliversOneNotification is the case a real Mobi over iOS
// produces: the whole response in a single notification with remaining=0, which
// is what TandemKit's BTResponseParser and pumpX2's PacketArrayList both key
// off (neither ever looks at a fragment's length).
func TestLargeMTUDeliversOneNotification(t *testing.T) {
	rig := newFaultRig(t)

	mustDo(t, rig.mux, http.MethodPut, "/api/transport", `{"att_mtu":247}`)

	if err := rig.route(t); err != nil {
		t.Fatalf("RouteMessage: %v", err)
	}

	got := rig.client.CollectNotifications(1, time.Second)
	if len(got) != 1 {
		t.Fatalf("central received %d notifications, want 1: %v", len(got), got)
	}

	fragment, err := hex.DecodeString(got[0])
	if err != nil {
		t.Fatalf("notification is not hex: %v", err)
	}
	if fragment[0]&0x0F != 0 {
		t.Errorf("remaining counter = %d, want 0 on a single-notification response", fragment[0]&0x0F)
	}
	if int(fragment[1]) != probeTxID {
		t.Errorf("txId = %d, want %d", fragment[1], probeTxID)
	}

	// Same message bytes, different framing: the payload must be exactly the
	// concatenation of the original fragments' payloads.
	var wantBody []byte
	for _, f := range rig.fragments {
		raw, err := hex.DecodeString(f)
		if err != nil {
			t.Fatalf("probe fragment is not hex: %v", err)
		}
		wantBody = append(wantBody, raw[2:]...)
	}
	if gotBody := fragment[2:]; hex.EncodeToString(gotBody) != hex.EncodeToString(wantBody) {
		t.Errorf("message bytes = %x, want %x", gotBody, wantBody)
	}

	// And the request log records what actually went out, not the encoder's
	// original framing -- a harness asserting on fragment counts reads this.
	resp := rig.lastResponseEntry(t)
	if len(resp.Fragments) != 1 || resp.FragmentsSent != 1 {
		t.Errorf("log recorded %d fragment(s), %d sent; want 1 and 1", len(resp.Fragments), resp.FragmentsSent)
	}
}

// TestMTUTooSmallForOneFragmentStillSplits checks the boundary: an MTU large
// enough for more than one 18-byte chunk but not the whole message still
// fragments, just into fewer pieces.
func TestMTUTooSmallForOneFragmentStillSplits(t *testing.T) {
	rig := newFaultRig(t)

	// 27-byte MTU -> 24-byte notifications -> 22 bytes of message each. The
	// probe's 34-byte body needs two.
	mustDo(t, rig.mux, http.MethodPut, "/api/transport", `{"att_mtu":27}`)

	if err := rig.route(t); err != nil {
		t.Fatalf("RouteMessage: %v", err)
	}

	got := rig.client.CollectNotifications(2, time.Second)
	if len(got) != 2 {
		t.Fatalf("central received %d fragments, want 2: %v", len(got), got)
	}
	for i, frag := range got {
		raw, err := hex.DecodeString(frag)
		if err != nil {
			t.Fatalf("fragment %d is not hex: %v", i, err)
		}
		if want := byte(len(got) - i - 1); raw[0]&0x0F != want {
			t.Errorf("fragment %d remaining = %d, want %d", i, raw[0]&0x0F, want)
		}
		if maxBytes := bluetooth.MaxNotificationBytes(27); len(raw) > maxBytes {
			t.Errorf("fragment %d is %d bytes, more than the %d a 27-byte MTU allows", i, len(raw), maxBytes)
		}
	}
}

// TestRejectsAnMTUOutsideTheSpecRange guards the endpoint's validation: a
// silently-accepted bad MTU would produce notifications no real link could
// carry.
func TestRejectsAnMTUOutsideTheSpecRange(t *testing.T) {
	rig := newFaultRig(t)

	for _, body := range []string{`{"att_mtu":22}`, `{"att_mtu":0}`, `{"att_mtu":518}`, `{"att_mtu":-1}`} {
		code, _ := do(t, rig.mux, http.MethodPut, "/api/transport", body)
		if code != http.StatusBadRequest {
			t.Errorf("PUT /api/transport %s = %d, want 400", body, code)
		}
	}

	// And the MTU is unchanged after the rejections.
	snapshot := mustDo(t, rig.mux, http.MethodGet, "/api/transport", "")
	if got := snapshot["att_mtu"]; got != float64(bluetooth.DefaultATTMTU) {
		t.Errorf("att_mtu = %v after rejected updates, want the %d default", got, bluetooth.DefaultATTMTU)
	}
}

// TestEmptyPutLeavesTheMTUAlone checks that a PUT with no att_mtu is a no-op
// rather than an attempt to set zero.
func TestEmptyPutLeavesTheMTUAlone(t *testing.T) {
	rig := newFaultRig(t)

	mustDo(t, rig.mux, http.MethodPut, "/api/transport", `{"att_mtu":185}`)
	snapshot := mustDo(t, rig.mux, http.MethodPut, "/api/transport", `{}`)
	if got := snapshot["att_mtu"]; got != float64(185) {
		t.Errorf("att_mtu = %v after an empty PUT, want the 185 already set", got)
	}
	if got := snapshot["max_notification_bytes"]; got != float64(182) {
		t.Errorf("max_notification_bytes = %v, want 182 (185 minus the 3-byte ATT header)", got)
	}
	if got := snapshot["max_message_bytes_per_notification"]; got != float64(180) {
		t.Errorf("max_message_bytes_per_notification = %v, want 180", got)
	}
}

// TestLargeMTUAlsoReframesNativeResponses covers the other encoder. Several
// messages a real pump sends cannot be built by the cliparser at all and are
// encoded natively instead -- HistoryLogStreamResponse among them -- and they
// go out through a separate send path that has to honor the same MTU.
func TestLargeMTUAlsoReframesNativeResponses(t *testing.T) {
	rig := newFaultRig(t)
	rig.client.Subscribe(bluetooth.HistoryLogCharUUID, true)

	records := [][]byte{make([]byte, 26), make([]byte, 26)}
	for i := range records {
		for j := range records[i] {
			records[i][j] = byte(i + 1)
		}
	}
	native, err := protocol.BuildHistoryLogStreamResponse(probeTxID, 1, records)
	if err != nil {
		t.Fatalf("BuildHistoryLogStreamResponse: %v", err)
	}
	if len(native.Fragments) < 2 {
		t.Fatalf("the test stream should need several fragments at the default MTU, got %d", len(native.Fragments))
	}

	mustDo(t, rig.mux, http.MethodPut, "/api/transport", `{"att_mtu":247}`)

	if err := rig.router.SendNative(native); err != nil {
		t.Fatalf("SendNative: %v", err)
	}

	got := rig.client.CollectNotifications(1, time.Second)
	if len(got) != 1 {
		t.Fatalf("central received %d notifications, want the whole stream in 1: %v", len(got), got)
	}

	// Same message bytes as the default framing carried, just in one packet.
	fragment, err := hex.DecodeString(got[0])
	if err != nil {
		t.Fatalf("notification is not hex: %v", err)
	}
	if fragment[0]&0x0F != 0 {
		t.Errorf("remaining counter = %d, want 0", fragment[0]&0x0F)
	}
	wantBody, err := protocol.MessageBodyFromFragmentsHex(native.PacketsHex())
	if err != nil {
		t.Fatalf("reassembling the native framing: %v", err)
	}
	if hex.EncodeToString(fragment[2:]) != hex.EncodeToString(wantBody) {
		t.Errorf("message bytes = %x, want %x", fragment[2:], wantBody)
	}
}
