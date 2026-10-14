package currentstatus_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCgmStatusV2Response(t *testing.T) {
	expected := currentstatus.NewCgmStatusV2Response(0, 0, 0, 0, 0, 0, 3, false)
	raw, _ := hex.DecodeString("0000000000000000000000000000000000000300")
	parsed := &currentstatus.CgmStatusV2Response{}
	if err := parsed.Parse(raw); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(parsed.Cargo(), expected.Cargo()) {
		t.Errorf("cargo %x, want %x", parsed.Cargo(), expected.Cargo())
	}
	if parsed.SessionState != currentstatus.CgmStatusV2SessionStateStatusStopped ||
		parsed.TransmitterBatteryStatus != currentstatus.CgmStatusV2TransmitterBatteryStatusUnavailable ||
		parsed.CgmSensorTypeID != 3 || parsed.CgmSensorType != currentstatus.CgmStatusV2SensorTypeDexcomG7 ||
		parsed.LastCalibrationTimestamp != 0 || parsed.SensorStartedTimestamp != 0 ||
		parsed.SessionDurationSeconds != 0 || parsed.SessionTimeRemainingSeconds != 0 || parsed.GracePeriod {
		t.Errorf("parsed %+v", parsed)
	}
}

func TestCgmStatusV2ResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CgmStatusV2Response")
}
