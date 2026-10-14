package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

// Each case's low alert is 80 mg/dL, repeating every 30 minutes.
func TestCGMGlucoseAlertSettingsResponse(t *testing.T) {
	for _, tc := range []struct {
		name                                   string
		hex                                    string
		txID                                   int
		highThreshold, highEnabled, highRepeat int32
		highDefaultBitmask                     int32
	}{
		{"high 200 mg/dL every hour", "00035b030cc800013c00055000011e00052a58", 3, 200, 1, 60, 5},
		{"high off", "00045b040cc800003c00015000011e0005af86", 4, 200, 0, 60, 1},
		{"high 200 mg/dL never repeating", "00055b050cc800010000015000011e000599d3", 5, 200, 1, 0, 1},
		{"high 250 mg/dL every 5 hours", "00065b060cfa00012c01005000011e00057eec", 6, 250, 1, 300, 0},
		{"high 120 mg/dL every 5 hours", "00075b070c7800012c01005000011e0005543b", 7, 120, 1, 300, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parity.TestMessage(t, tc.hex, tc.txID, 1, messages.CurrentStatus,
				currentstatus.NewCGMGlucoseAlertSettingsResponse(tc.highThreshold, tc.highEnabled, tc.highRepeat, tc.highDefaultBitmask, 80, 1, 30, 5))
		})
	}
}

func TestCGMGlucoseAlertSettingsResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CGMGlucoseAlertSettingsResponse")
}
