package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCurrentBatteryV1Response(t *testing.T) {
	for _, tc := range []struct {
		name     string
		hex      string
		txID     int
		abc, ibc int32
	}{
		{"99/100", "0003350302636452b9", 3, 99, 100},
		{"full", "00043504026464e871", 4, 100, 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed := parity.TestMessage(t, tc.hex, tc.txID, 1, messages.CurrentStatus,
				currentstatus.NewCurrentBatteryV1Response(tc.abc, tc.ibc)).(*currentstatus.CurrentBatteryV1Response)
			if parsed.BatteryPercent() != tc.ibc {
				t.Errorf("battery %d%%, want %d%%", parsed.BatteryPercent(), tc.ibc)
			}
		})
	}
}

func TestCurrentBatteryV1ResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CurrentBatteryV1Response")
}
