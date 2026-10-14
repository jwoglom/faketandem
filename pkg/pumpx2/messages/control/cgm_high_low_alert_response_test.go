package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCgmHighLowAlertResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CgmHighLowAlertResponse")
}
