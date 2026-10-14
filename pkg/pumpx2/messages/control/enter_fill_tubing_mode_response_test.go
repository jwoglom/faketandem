package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestEnterFillTubingModeResponse(t *testing.T) {
	parity.TestMessage(t, "007a957a1900fea3ee1fe559e33c9fdc97594748e9c9631ee2db4458683f3412", 122, 1, messages.Control, control.NewEnterFillTubingModeResponse(0))
}

func TestEnterFillTubingModeResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "EnterFillTubingModeResponse")
}
