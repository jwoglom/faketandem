package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCurrentBasalStatusRequest(t *testing.T) {
	parity.TestMessage(t, "0003280300a8b6", 3, 1, messages.CurrentStatus, currentstatus.NewCurrentBasalStatusRequest())
}

func TestCurrentBasalStatusRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CurrentBasalStatusRequest")
}
