package controlstream_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestNonexistentExitFillTubingModeStateStreamRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "NonexistentExitFillTubingModeStateStreamRequest")
}
