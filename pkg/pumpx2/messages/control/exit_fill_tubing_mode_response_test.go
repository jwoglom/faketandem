package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestExitFillTubingModeResponse(t *testing.T) {
	parity.TestMessage(t, "009197911900aec8402021f71d880600f40c8f92f96bbca87dbea12d73ad213b", 0x91, 1, messages.Control, control.NewExitFillTubingModeResponse(0))
}

func TestExitFillTubingModeResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ExitFillTubingModeResponse")
}
