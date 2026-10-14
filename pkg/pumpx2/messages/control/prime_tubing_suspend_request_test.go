package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestPrimeTubingSuspendRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "PrimeTubingSuspendRequest")
}
