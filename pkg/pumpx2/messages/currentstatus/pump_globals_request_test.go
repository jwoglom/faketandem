package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestPumpGlobalsRequest(t *testing.T) {
	parity.TestMessage(t, "0003560300a175", 3, 1, messages.CurrentStatus, currentstatus.NewPumpGlobalsRequest())
}

func TestPumpGlobalsRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "PumpGlobalsRequest")
}
