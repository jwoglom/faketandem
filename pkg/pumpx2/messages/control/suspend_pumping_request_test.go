package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSuspendPumpingRequest(t *testing.T) {
	parity.TestMessage(t, "01659c651838eaee1f67bfcaf05f7c5e09412c65", 101, 1, messages.Control, control.NewSuspendPumpingRequest(),
		"00651de503c71694d7e46bce00")
}

func TestSuspendPumpingRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SuspendPumpingRequest")
}
