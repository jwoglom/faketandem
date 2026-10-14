package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetAutoOffAlertResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetAutoOffAlertResponse")
}
