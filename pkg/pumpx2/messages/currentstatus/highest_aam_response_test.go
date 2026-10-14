package currentstatus_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func newHighestAamResponse(t *testing.T, aamID, faultID int64, remaining []byte) *currentstatus.HighestAamResponse {
	t.Helper()
	m, err := currentstatus.NewHighestAamResponse(aamID, faultID, remaining)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestHighestAamResponseStuckVibratorMotor(t *testing.T) {
	parsed := parity.TestMessage(t, "000779070b0c0000007120000004ffffd6a6", 7, 1, messages.CurrentStatus,
		newHighestAamResponse(t, 12, 0x2071, []byte{4, 0xff, 0xff})).(*currentstatus.HighestAamResponse)
	if parsed.AamID != 12 || parsed.FaultID != 0x2071 || parsed.FaultIDAsHex() != "2071" {
		t.Errorf("parsed %+v", parsed)
	}
	if parsed.ErrorString != "12-0x2071" || !parsed.HasMalfunction() {
		t.Errorf("error string %q, malfunction %t", parsed.ErrorString, parsed.HasMalfunction())
	}
}

func TestHighestAamResponseIgnoresCodesThatComeWithAnAlarm(t *testing.T) {
	for _, tc := range []struct {
		name           string
		hex            string
		txID           int
		aamID, faultID int64
		remaining      []byte
		alarm          currentstatus.AlarmResponseType
	}{
		{"3-0x2026 with a pump reset", "001079100b030000002620000002ffff2ab0", 16, 3, 0x2026, []byte{2, 0xff, 0xff},
			currentstatus.AlarmResponseTypePumpResetAlarm},
		{"18-0x2077 with resume pump", "005979590b120000007720000002030a5792", 89, 18, 0x2077, []byte{2, 3, 10},
			currentstatus.AlarmResponseTypeResumePumpAlarm},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed := parity.TestMessage(t, tc.hex, tc.txID, 1, messages.CurrentStatus,
				newHighestAamResponse(t, tc.aamID, tc.faultID, tc.remaining)).(*currentstatus.HighestAamResponse)
			if parsed.AamID != tc.aamID || parsed.FaultID != tc.faultID {
				t.Errorf("parsed %+v", parsed)
			}
			if parsed.HasMalfunction(currentstatus.NewAlarmStatusResponseFromAlarms(tc.alarm)) {
				t.Error("reported as a malfunction")
			}
		})
	}
}

func TestHighestAamResponseIgnoresTheIDOfAnAlarmReadAlongside(t *testing.T) {
	m := newHighestAamResponse(t, int64(currentstatus.AlarmResponseTypeOcclusionAlarm), 0x1234, []byte{1, 2, 3})
	if !m.HasMalfunction() || m.Size() != 1 {
		t.Error("not a malfunction on its own")
	}
	if m.HasMalfunction(currentstatus.NewAlarmStatusResponseFromAlarms(currentstatus.AlarmResponseTypeOcclusionAlarm)) {
		t.Error("a malfunction alongside the occlusion alarm with its ID")
	}
}

func TestHighestAamResponseParsesTenBytes(t *testing.T) {
	parsed := currentstatus.NewEmptyHighestAamResponse()
	if err := parsed.Parse([]byte{12, 0, 0, 0, 0x71, 0x20, 0, 0, 4, 0xff}); err != nil {
		t.Fatal(err)
	}
	if parsed.AamID != 12 || parsed.FaultID != 0x2071 || parsed.FaultIDAsHex() != "2071" || parsed.ErrorString != "12-0x2071" {
		t.Errorf("parsed %+v", parsed)
	}
	if !bytes.Equal(parsed.Remaining, []byte{4, 0xff}) {
		t.Errorf("remaining %x", parsed.Remaining)
	}
}

func TestHighestAamResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "HighestAamResponse")
}
