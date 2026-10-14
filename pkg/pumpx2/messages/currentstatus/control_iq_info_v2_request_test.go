package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestControlIQInfoV2RequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ControlIQInfoV2Request")
}
