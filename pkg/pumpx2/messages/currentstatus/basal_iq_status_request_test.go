package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestBasalIQStatusRequest(t *testing.T) {
	parity.TestMessage(t, "000570050061eb", 5, 1, messages.CurrentStatus, currentstatus.NewBasalIQStatusRequest())
}

func TestBasalIQStatusRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "BasalIQStatusRequest")
}
