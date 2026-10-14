package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestEnterChangeCartridgeModeRequest(t *testing.T) {
	parity.TestMessage(t, "01369036181d142820db51e5fa626a7df87fc2cf", 54, 1, messages.Control, control.NewEnterChangeCartridgeModeRequest(), "003621097bdd2395333e5d7d63")
}

func TestEnterChangeCartridgeModeRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "EnterChangeCartridgeModeRequest")
}
