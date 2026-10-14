package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestRenameIDPResponse(t *testing.T) {
	parity.TestMessage(t, "00d6a9d61a0002b5023e20a9f6db74a433fd5d0f2abc3091f359c4b379c92a4ff6", -42, 1, messages.Control,
		control.NewRenameIDPResponse(0, 2))
}

func TestRenameIDPResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "RenameIDPResponse")
}
