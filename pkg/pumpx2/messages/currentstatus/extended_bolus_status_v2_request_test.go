package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestExtendedBolusStatusV2RequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ExtendedBolusStatusV2Request")
}
