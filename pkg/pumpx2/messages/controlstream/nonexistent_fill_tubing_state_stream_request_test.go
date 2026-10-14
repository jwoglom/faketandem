package controlstream_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestNonexistentFillTubingStateStreamRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "NonexistentFillTubingStateStreamRequest")
}
