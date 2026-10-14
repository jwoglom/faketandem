package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetActiveIDPResponse(t *testing.T) {
	parity.TestMessage(t, "0019ed19190036033e20eb7b1d7dc0f2aad134ccab6029516c36a4d910b55778", 25, 1, messages.Control,
		control.NewSetActiveIDPResponse(0))
}

func TestSetActiveIDPResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetActiveIDPResponse")
}
