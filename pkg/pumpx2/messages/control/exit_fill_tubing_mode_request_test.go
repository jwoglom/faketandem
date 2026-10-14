package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestExitFillTubingModeRequest(t *testing.T) {
	parity.TestMessage(t, "01919691181b0f4120ad6869443532522e06f02a", 0x91, 1, messages.Control, control.NewExitFillTubingModeRequest(), "0091687757e745d1f795a3295f")
}

func TestExitFillTubingModeRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ExitFillTubingModeRequest")
}
