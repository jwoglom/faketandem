package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestInitiateBolusResponse(t *testing.T) {
	for _, tc := range []struct {
		name           string
		timeSinceReset string
		hex            string
		txID           int
		bolusID        int32
	}{
		{"bolus 10650", "461510642", "00399f391e009a29000000f717821b19f3e1e5f2e0950d2ede3572f325e6f400b019c37ce3", 57, 10650},
		{"bolus 10652", "461589180", "003e9f3e1e009c29000000e64a831b494e667a54ee8b35042d93ad4af8c612963d76b86f60", 62, 10652},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tester := parity.MessageTester{Env: messages.Env{PairingCode: "6VeDeRAL5DCigGw2", TimeSinceReset: tc.timeSinceReset}}
			parsed := tester.Test(t, tc.hex, tc.txID, 1, messages.Control, control.NewInitiateBolusResponse(0, tc.bolusID, 0)).(*control.InitiateBolusResponse)
			if !parsed.WasBolusInitiated() {
				t.Error("the bolus was not initiated")
			}
		})
	}
}

func TestInitiateBolusResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "InitiateBolusResponse")
}
