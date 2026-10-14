package controlstream_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestNonexistentPrimeNudgeStateStreamRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "NonexistentPrimeNudgeStateStreamRequest")
}
