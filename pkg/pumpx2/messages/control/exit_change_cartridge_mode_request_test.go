package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestExitChangeCartridgeModeRequest(t *testing.T) {
	parity.TestMessage(t, "0189928918e40e4120bd5bce64b42a9df62d8cae", 0x89, 1, messages.Control, control.NewExitChangeCartridgeModeRequest(), "0089e8d1e8325cc6cacddc7790")
}

func TestExitChangeCartridgeModeRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ExitChangeCartridgeModeRequest")
}
