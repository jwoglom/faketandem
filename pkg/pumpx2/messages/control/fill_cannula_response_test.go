package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestFillCannulaResponse(t *testing.T) {
	parity.TestMessage(t, "006d996d1900d4ce2720d38b10afb8f57ab650c6682172ef1f0af1e8d4e59e0c", 109, 1, messages.Control,
		control.NewFillCannulaResponse(0))
}

func TestFillCannulaResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "FillCannulaResponse")
}
