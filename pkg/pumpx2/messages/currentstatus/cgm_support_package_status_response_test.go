package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCgmSupportPackageStatusResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CgmSupportPackageStatusResponse")
}
