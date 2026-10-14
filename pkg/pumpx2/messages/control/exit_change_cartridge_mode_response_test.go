package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestExitChangeCartridgeModeResponse(t *testing.T) {
	parity.TestMessage(t, "00899389190078c840201a521d880f1dfa4fd153a975571060d9098181614321", 0x89, 1, messages.Control, control.NewExitChangeCartridgeModeResponse(0))
}

func TestExitChangeCartridgeModeResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ExitChangeCartridgeModeResponse")
}
