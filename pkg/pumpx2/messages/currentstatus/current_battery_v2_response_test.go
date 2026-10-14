package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCurrentBatteryV2Response(t *testing.T) {
	for _, tc := range []struct {
		name               string
		hex                string
		txID               int
		abc, ibc, charging int32
	}{
		{"fully charged", "000d910d0b6464000000000000000000eb4f", 13, 100, 100, 0},
		{"displays 10%", "000691060b1b0a00000000000000000056c9", 6, 27, 10, 0},
		{"displays 10%, charging", "000791070b1b0a0100000000000000001667", 7, 27, 10, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed := parity.TestMessage(t, tc.hex, tc.txID, 1, messages.CurrentStatus,
				currentstatus.NewCurrentBatteryV2Response(tc.abc, tc.ibc, tc.charging, 0, 0, 0, 0)).(*currentstatus.CurrentBatteryV2Response)
			if parsed.BatteryPercent() != tc.ibc || parsed.Charging() != (tc.charging == 1) {
				t.Errorf("battery %d%%, charging %t", parsed.BatteryPercent(), parsed.Charging())
			}
		})
	}
}

func TestCurrentBatteryV2ResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CurrentBatteryV2Response")
}
