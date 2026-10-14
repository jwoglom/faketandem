package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestDisconnectPumpResponse(t *testing.T) {
	parity.TestMessage(t, "00debfde1900000000003fad35823cf021e2dad26d3f485cd5b4557dfe88ef21", 0xde, 1, messages.Control, control.NewDisconnectPumpResponse(0))
}

func TestDisconnectPumpResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "DisconnectPumpResponse")
}
