package currentstatus_test

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

// The pump sets bits pumpX2 does not name, which parse to no feature.
func TestPumpFeaturesV1Response(t *testing.T) {
	for _, tc := range []struct {
		name     string
		hex      string
		txID     int
		intMap   int64
		features []currentstatus.PumpFeatureType
	}{
		{"t:slim X2", "00034f03089add2002000000007f08", 3, 35708314, []currentstatus.PumpFeatureType{
			currentstatus.PumpFeatureDexcomG6Supported, currentstatus.PumpFeatureControlIQSupported, currentstatus.PumpFeatureAutoPopSupported,
		}},
		{"t:slim X2 7.4", "00044f04089add2426000000009693", 4, 639950234, []currentstatus.PumpFeatureType{
			currentstatus.PumpFeatureDexcomG6Supported, currentstatus.PumpFeatureControlIQSupported, currentstatus.PumpFeatureBasalLimitSupported,
			currentstatus.PumpFeatureAutoPopSupported, currentstatus.PumpFeaturePumpSettingsInIDPGUISupported,
		}},
		{"t:slim X2 7.6", "00034f03089add2476000000005e9a", 3, 1982127514, []currentstatus.PumpFeatureType{
			currentstatus.PumpFeatureDexcomG6Supported, currentstatus.PumpFeatureControlIQSupported, currentstatus.PumpFeatureBasalLimitSupported,
			currentstatus.PumpFeatureAutoPopSupported, currentstatus.PumpFeatureBLEPumpControlSupported, currentstatus.PumpFeaturePumpSettingsInIDPGUISupported,
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := currentstatus.NewPumpFeaturesV1Response(big.NewInt(tc.intMap))
			if err != nil {
				t.Fatal(err)
			}
			parsed := parity.TestMessage(t, tc.hex, tc.txID, 1, messages.CurrentStatus, expected).(*currentstatus.PumpFeaturesV1Response)
			if !reflect.DeepEqual(parsed.Features, tc.features) {
				t.Errorf("features %v, want %v", parsed.Features, tc.features)
			}
		})
	}
}

func TestPumpFeaturesV1ResponseOfAMobi(t *testing.T) {
	parsed, err := currentstatus.NewPumpFeaturesV1ResponseFromCargo([]byte{18, 21, 101, 118, 0, 0, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	want := []currentstatus.PumpFeatureType{
		currentstatus.PumpFeatureDexcomG6Supported, currentstatus.PumpFeatureControlIQSupported, currentstatus.PumpFeatureWombatSupported,
		currentstatus.PumpFeatureBasalLimitSupported, currentstatus.PumpFeatureAutoPopSupported, currentstatus.PumpFeatureBLEPumpControlSupported,
		currentstatus.PumpFeaturePumpSettingsInIDPGUISupported,
	}
	if !reflect.DeepEqual(parsed.Features, want) {
		t.Errorf("features %v, want %v", parsed.Features, want)
	}
	if parsed.IntMap.Cmp(big.NewInt(1986336018)) != 0 {
		t.Errorf("intMap %v, want 1986336018", parsed.IntMap)
	}
}

func TestPumpFeaturesV1ResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "PumpFeaturesV1Response")
}
