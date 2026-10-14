package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCGMStatusCRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CGMStatusCRequest")
}
