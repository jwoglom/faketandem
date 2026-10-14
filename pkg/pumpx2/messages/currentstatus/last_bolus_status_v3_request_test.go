package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestLastBolusStatusV3RequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "LastBolusStatusV3Request")
}
