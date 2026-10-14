package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestBolusPermissionRequest(t *testing.T) {
	tester := parity.MessageTester{Env: messages.Env{PairingCode: "6VeDeRAL5DCigGw2", TimeSinceReset: "461457713"}}
	tester.Test(t, "0134a234181d47811bc64bcd072fc978e4842046", 52, 2, messages.Control,
		control.NewBolusPermissionRequest(), "003426b62ba72612b4f04978ac")
}

func TestBolusPermissionRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "BolusPermissionRequest")
}
