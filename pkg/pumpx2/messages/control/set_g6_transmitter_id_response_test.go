package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetG6TransmitterIDResponse(t *testing.T) {
	parity.TestMessage(t, "0021b121190085c3f21f51e3fabfa0a59cd348aed259b44201903b33d9bead20", 33, 1, messages.Control,
		control.NewSetG6TransmitterIDResponse(0))
}

func TestSetG6TransmitterIDResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetG6TransmitterIdResponse")
}
