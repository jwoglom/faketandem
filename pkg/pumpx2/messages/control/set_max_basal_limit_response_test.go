package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetMaxBasalLimitResponse(t *testing.T) {
	parity.TestMessage(t, "00758975190029e34920e1e6145d9ec4ced8f764873318b61dac15b7e3fee55e", 117, 1, messages.Control, control.NewSetMaxBasalLimitResponse(0))
}

func TestSetMaxBasalLimitResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetMaxBasalLimitResponse")
}
