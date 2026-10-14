package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestEnterFillTubingModeRequest(t *testing.T) {
	parity.TestMessage(t, "017a947a1851eaee1f5595738c382cc5dc5552a2", 122, 1, messages.Control, control.NewEnterFillTubingModeRequest(), "007a4aa8482c322b5fd69805e8")
}

func TestEnterFillTubingModeRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "EnterFillTubingModeRequest")
}
