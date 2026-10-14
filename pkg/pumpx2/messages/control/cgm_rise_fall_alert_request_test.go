package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCgmRiseFallAlertRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CgmRiseFallAlertRequest")
}
