package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestEnterChangeCartridgeModeResponse(t *testing.T) {
	parity.TestMessage(t, "003691361900e2cd272051bfc76d9666fb6bbe8b89627199c182617b1f6fe440", 54, 1, messages.Control, control.NewEnterChangeCartridgeModeResponse(0))
}

func TestEnterChangeCartridgeModeResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "EnterChangeCartridgeModeResponse")
}
