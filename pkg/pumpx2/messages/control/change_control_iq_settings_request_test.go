package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestChangeControlIQSettingsRequest(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled bool
		txID    int
		packets []string
	}{
		{"turn on", true, 87, []string{"0157ca571e019600014b0127eaee1fe2a83a5338", "00578298cbe960ee83e35b3d3fde85b38d7eba"}},
		{"turn off", false, 92, []string{"015cca5c1e009600014b012ceaee1f556e63dee5", "005ce9516bf5a6d77eb884dc557e841663bea5"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed := parity.TestMessage(t, tc.packets[0], tc.txID, 1, messages.Control,
				control.NewChangeControlIQSettingsRequest(tc.enabled, 150, 75), tc.packets[1:]...).(*control.ChangeControlIQSettingsRequest)
			if parsed.Enabled != tc.enabled || parsed.WeightLbs != 150 || parsed.TotalDailyInsulinUnits != 75 {
				t.Errorf("parsed %+v", *parsed)
			}
		})
	}
}

func TestChangeControlIQSettingsRequestReadsALargeTotalDailyInsulinAsNegative(t *testing.T) {
	if got := control.NewChangeControlIQSettingsRequest(true, 150, 200).TotalDailyInsulinUnits; got != -56 {
		t.Errorf("200 units read back as %d, want -56", got)
	}
}

func TestChangeControlIQSettingsRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ChangeControlIQSettingsRequest")
}
