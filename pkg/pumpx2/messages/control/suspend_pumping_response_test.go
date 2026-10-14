package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSuspendPumpingResponse(t *testing.T) {
	parity.TestMessage(t, "00659d651900e6a3ee1f4114c9f737f36d02bf6794cc45047602f27e1da5996a", 101, 1, messages.Control, control.NewSuspendPumpingResponse(0))
}

func TestSuspendPumpingResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SuspendPumpingResponse")
}
