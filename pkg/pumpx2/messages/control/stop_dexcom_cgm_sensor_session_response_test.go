package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestStopDexcomCGMSensorSessionResponse(t *testing.T) {
	parity.TestMessage(t, "005ab55a19002cc7f21f7fc48fc9d4bfe3feca73675911cca9f1b3a96caba40c", 90, 1, messages.Control, control.NewStopDexcomCGMSensorSessionResponse(0))
}

func TestStopDexcomCGMSensorSessionResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "StopDexcomCGMSensorSessionResponse")
}
