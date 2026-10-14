package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCgmOutOfRangeAlertRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CgmOutOfRangeAlertRequest")
}
