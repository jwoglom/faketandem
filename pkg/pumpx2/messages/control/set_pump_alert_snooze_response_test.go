package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetPumpAlertSnoozeResponse(t *testing.T) {
	parity.TestMessage(t, "0031d5311900ba80682053558873893a6be84fccd0aa1a79beb7a1eb118a9cb8", 49, 1, messages.Control, control.NewSetPumpAlertSnoozeResponse(0))
}

func TestSetPumpAlertSnoozeResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetPumpAlertSnoozeResponse")
}
