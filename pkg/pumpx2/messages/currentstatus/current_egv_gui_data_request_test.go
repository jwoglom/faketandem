package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCurrentEGVGuiDataRequest(t *testing.T) {
	parity.TestMessage(t, "00032203006971", 3, 1, messages.CurrentStatus, currentstatus.NewCurrentEGVGuiDataRequest())
}

func TestCurrentEGVGuiDataRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CurrentEGVGuiDataRequest")
}
