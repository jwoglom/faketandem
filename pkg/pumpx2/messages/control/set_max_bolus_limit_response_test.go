package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetMaxBolusLimitResponse(t *testing.T) {
	parity.TestMessage(t, "0061876119007de14920248f42cba27ec99fd6e97b48b5e464ad29691821c4eb", 97, 1, messages.Control, control.NewSetMaxBolusLimitResponse(0))
}

func TestSetMaxBolusLimitResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetMaxBolusLimitResponse")
}
