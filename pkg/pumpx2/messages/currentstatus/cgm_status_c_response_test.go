package currentstatus_test

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCGMStatusCResponse(t *testing.T) {
	expected := currentstatus.NewCGMStatusCResponse(0, 0, 0, 0, 0, 0, 3, false, 0, 0)
	raw, _ := hex.DecodeString("000000000000000000000000000000000000030000000000")
	parsed := &currentstatus.CGMStatusCResponse{}
	if err := parsed.Parse(raw); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(parsed.Cargo(), expected.Cargo()) {
		t.Errorf("cargo %x, want %x", parsed.Cargo(), expected.Cargo())
	}
	want := currentstatus.CGMStatusCResponse{
		SessionState:             currentstatus.CGMStatusCSessionStateStatusStopped,
		TransmitterBatteryStatus: currentstatus.CGMStatusCTransmitterBatteryStatusUnavailable,
		CgmSensorTypeID:          3,
		CgmSensorType:            currentstatus.CGMStatusCSensorTypeDexcomG7,
	}
	want.SetCargo(parsed.Cargo())
	if !reflect.DeepEqual(*parsed, want) {
		t.Errorf("parsed %+v, want %+v", *parsed, want)
	}
}

func TestCGMStatusCResponseActiveSessionWithWarmup(t *testing.T) {
	expected := currentstatus.NewCGMStatusCResponse(2, 500000000, 500000100, 3, 864000, 863100, 3, true, 1800, 900)
	parsed := &currentstatus.CGMStatusCResponse{}
	if err := parsed.Parse(expected.Cargo()); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Cargo()) != 24 {
		t.Errorf("%d bytes, want 24", len(parsed.Cargo()))
	}
	want := currentstatus.CGMStatusCResponse{
		SessionStateID:                 2,
		SessionState:                   currentstatus.CGMStatusCSessionStateSessionActive,
		LastCalibrationTimestamp:       500000000,
		SensorStartedTimestamp:         500000100,
		TransmitterBatteryStatusID:     3,
		TransmitterBatteryStatus:       currentstatus.CGMStatusCTransmitterBatteryStatusOkay,
		SessionDurationSeconds:         864000,
		SessionTimeRemainingSeconds:    863100,
		CgmSensorTypeID:                3,
		CgmSensorType:                  currentstatus.CGMStatusCSensorTypeDexcomG7,
		GracePeriod:                    true,
		SensorWarmupDurationSeconds:    1800,
		RemainingWarmupDurationSeconds: 900,
	}
	want.SetCargo(parsed.Cargo())
	if !reflect.DeepEqual(*parsed, want) {
		t.Errorf("parsed %+v, want %+v", *parsed, want)
	}
}

func TestCGMStatusCResponseReadsWarmupUnsigned(t *testing.T) {
	m := currentstatus.NewCGMStatusCResponse(1, 0, 0, 0, 0, 0, 1, false, 65535, 65535)
	if m.SensorWarmupDurationSeconds != 65535 || m.RemainingWarmupDurationSeconds != 65535 {
		t.Errorf("warmup %d, %d remaining, want 65535", m.SensorWarmupDurationSeconds, m.RemainingWarmupDurationSeconds)
	}
}

func TestCGMStatusCResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CGMStatusCResponse")
}
