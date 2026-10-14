package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestPumpVersionBRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "PumpVersionBRequest")
}
