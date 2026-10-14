package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestLastBolusStatusRequest(t *testing.T) {
	parity.TestMessage(t, "0005300500ccf6", 5, 1, messages.CurrentStatus, currentstatus.NewLastBolusStatusRequest())
}

func TestLastBolusStatusRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "LastBolusStatusRequest")
}
