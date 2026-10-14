package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCurrentEgvGuiDataV2ResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CurrentEgvGuiDataV2Response")
}
