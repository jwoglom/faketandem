package currentstatus_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCGMStatusResponse(t *testing.T) {
	expected := currentstatus.NewCGMStatusResponse(0, 0, 0, 0)
	raw, _ := hex.DecodeString("00000000000000000000")
	parsed := &currentstatus.CGMStatusResponse{}
	if err := parsed.Parse(raw); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(parsed.Cargo(), expected.Cargo()) {
		t.Errorf("cargo %x, want %x", parsed.Cargo(), expected.Cargo())
	}
	if parsed.SessionStateID != 0 || parsed.SessionState == nil || *parsed.SessionState != currentstatus.CGMStatusSessionStateSessionStopped ||
		parsed.LastCalibrationTimestamp != 0 || parsed.SensorStartedTimestamp != 0 || parsed.TransmitterBatteryStatusID != 0 ||
		parsed.TransmitterBatteryStatus == nil || *parsed.TransmitterBatteryStatus != currentstatus.CGMStatusTransmitterBatteryStatusUnavailable {
		t.Errorf("parsed %+v", parsed)
	}
}

func TestCGMStatusResponseLeavesAnUnknownStateNil(t *testing.T) {
	m := currentstatus.NewCGMStatusResponse(9, 0, 0, -1)
	if m.SessionState != nil || m.TransmitterBatteryStatus != nil {
		t.Errorf("session state %v, battery %v, want nil", m.SessionState, m.TransmitterBatteryStatus)
	}
}

func TestCGMStatusResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CGMStatusResponse")
}
