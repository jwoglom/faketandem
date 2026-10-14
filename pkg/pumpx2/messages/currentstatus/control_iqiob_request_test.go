package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestControlIQIOBRequest(t *testing.T) {
	parity.TestMessage(t, "00046c040052ee", 4, 1, messages.CurrentStatus, currentstatus.NewControlIQIOBRequest())
}

func TestControlIQIOBRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ControlIQIOBRequest")
}
