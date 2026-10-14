package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCgmHighLowAlertRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CgmHighLowAlertRequest")
}
