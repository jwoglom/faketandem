package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestBolusPermissionReleaseRequest(t *testing.T) {
	tester := parity.MessageTester{Env: messages.Env{PairingCode: "6VeDeRAL5DCigGw2", TimeSinceReset: "461710079"}}
	tester.Test(t, "013af03a1cb42900000023851b3c39b657fe391e", 58, 2, messages.Control,
		control.NewBolusPermissionReleaseRequest(10676), "003ac14d83666a2599ae79a30e5d9b459a")
}

func TestBolusPermissionReleaseRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "BolusPermissionReleaseRequest")
}
