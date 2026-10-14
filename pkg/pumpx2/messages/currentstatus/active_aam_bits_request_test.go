package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestActiveAamBitsRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ActiveAamBitsRequest")
}
