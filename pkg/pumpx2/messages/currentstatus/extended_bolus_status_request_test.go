package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestExtendedBolusStatusRequest(t *testing.T) {
	parity.TestMessage(t, "00062e0600fdfb", 6, 1, messages.CurrentStatus, currentstatus.NewExtendedBolusStatusRequest())
}

func TestExtendedBolusStatusRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ExtendedBolusStatusRequest")
}
