package handler

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"
)

// The IDP requests are handed over as raw cargo, which is what cliparser's parse carries under
// "cargo"; the answers are parsed back through the real cliparser, as a driver would decode them.
//
// Skipped unless FAKETANDEM_TEST_CLIPARSER_JAR is set.

func idpAnswer(t *testing.T, bridge *pumpx2.Bridge, ps *state.PumpState, request string, cargo []byte) map[string]interface{} {
	t.Helper()

	h := handlerFor(t, bridge, request)
	msg := &pumpx2.ParsedMessage{TxID: 7, MessageType: request, Cargo: map[string]interface{}{"cargo": hex.EncodeToString(cargo)}}
	resp, err := h.HandleMessage(msg, ps)
	if err != nil {
		t.Fatalf("%s handler failed: %v", request, err)
	}
	char := bluetooth.CharCurrentStatus
	if request == "SetIDPSegmentRequest" || request == "CreateIDPRequest" {
		char = bluetooth.CharControl
	}
	parsed, err := bridge.ParseMessage(char, resp.ResponseMessage.Packets)
	if err != nil {
		t.Fatalf("could not parse the %s answer back: %v", request, err)
	}
	return parsed.Cargo
}

func answerInt(t *testing.T, answer map[string]interface{}, key string) int {
	t.Helper()
	var v int
	if _, err := fmt.Sscan(fmt.Sprint(answer[key]), &v); err != nil {
		t.Fatalf("answer has no numeric %s: %v", key, answer)
	}
	return v
}

func setIDPSegmentCargo(idpID, segmentIndex int, op state.IDPSegmentOperation, startTime, basalRate int) []byte {
	cargo := make([]byte, 17)
	cargo[0], cargo[2], cargo[3] = byte(idpID), byte(segmentIndex), byte(op)
	binary.LittleEndian.PutUint16(cargo[4:], uint16(startTime))
	binary.LittleEndian.PutUint16(cargo[6:], uint16(basalRate))
	binary.LittleEndian.PutUint32(cargo[8:], 10000)
	binary.LittleEndian.PutUint16(cargo[12:], 110)
	binary.LittleEndian.PutUint16(cargo[14:], 50)
	cargo[16] = 31
	return cargo
}

func createIDPCargo(name string, basalRate int) []byte {
	cargo := make([]byte, 35)
	copy(cargo, name)
	binary.LittleEndian.PutUint32(cargo[17:], 10000)
	binary.LittleEndian.PutUint16(cargo[23:], uint16(basalRate))
	binary.LittleEndian.PutUint16(cargo[25:], 110)
	binary.LittleEndian.PutUint16(cargo[27:], 40)
	binary.LittleEndian.PutUint16(cargo[29:], 240)
	cargo[31], cargo[32], cargo[33] = 31, 5, 0xFF
	return cargo
}

func TestIDPWritesAreReadBackThroughTheProfileRequests(t *testing.T) {
	bridge := testBridge(t)
	ps := state.NewPumpState()
	ps.SetIDPProfiles(nil)

	status := idpAnswer(t, bridge, ps, "ProfileStatusRequest", nil)
	if n := answerInt(t, status, "numberOfProfiles"); n != 0 {
		t.Fatalf("an empty pump reports %d profiles", n)
	}

	created := idpAnswer(t, bridge, ps, "CreateIDPRequest", createIDPCargo("TandemKitProfile", 650))
	id := answerInt(t, created, "newIdpId")
	if s := answerInt(t, created, "status"); s != 0 {
		t.Fatalf("CreateIDP answered status %d", s)
	}

	status = idpAnswer(t, bridge, ps, "ProfileStatusRequest", nil)
	if n, slot0 := answerInt(t, status, "numberOfProfiles"), answerInt(t, status, "idpSlot0Id"); n != 1 || slot0 != id {
		t.Fatalf("after CreateIDP: %d profiles, slot 0 holds %d, want 1 and %d", n, slot0, id)
	}

	write := idpAnswer(t, bridge, ps, "SetIDPSegmentRequest", setIDPSegmentCargo(id, 0, state.IDPCreateSegment, 360, 500))
	if s := answerInt(t, write, "status"); s != 0 {
		t.Fatalf("creating a segment answered status %d", s)
	}
	collision := idpAnswer(t, bridge, ps, "SetIDPSegmentRequest", setIDPSegmentCargo(id, 0, state.IDPCreateSegment, 360, 400))
	if s := answerInt(t, collision, "status"); s == 0 {
		t.Fatal("a segment onto a taken start time was accepted")
	}

	settings := idpAnswer(t, bridge, ps, "IDPSettingsRequest", []byte{byte(id)})
	if n := answerInt(t, settings, "numberOfProfileSegments"); n != 2 {
		t.Fatalf("IDPSettings reports %d segments, want 2", n)
	}
	for index, want := range [][2]int{{0, 650}, {360, 500}} {
		segment := idpAnswer(t, bridge, ps, "IDPSegmentRequest", []byte{byte(id), byte(index)})
		if got := [2]int{answerInt(t, segment, "profileStartTime"), answerInt(t, segment, "profileBasalRate")}; got != want {
			t.Errorf("segment %d reads back as %v, want %v", index, got, want)
		}
	}
}
