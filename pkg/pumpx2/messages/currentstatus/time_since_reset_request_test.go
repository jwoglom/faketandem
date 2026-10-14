package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestTimeSinceResetRequest(t *testing.T) {
	parity.TestMessage(t, "0003360300caee", 3, 1, messages.CurrentStatus, currentstatus.NewTimeSinceResetRequest())
}

func TestTimeSinceResetRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "TimeSinceResetRequest")
}
