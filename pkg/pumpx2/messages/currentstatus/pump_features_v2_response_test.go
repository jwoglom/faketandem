package currentstatus_test

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func parsePumpFeaturesV2Response(t *testing.T, cargoHex string) *currentstatus.PumpFeaturesV2Response {
	t.Helper()
	cargo, err := hex.DecodeString(cargoHex)
	if err != nil {
		t.Fatal(err)
	}
	m := &currentstatus.PumpFeaturesV2Response{}
	if err := m.Parse(cargo); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestPumpFeaturesV2Response(t *testing.T) {
	for _, tc := range []struct {
		cargo   string
		status  int32
		index   currentstatus.SupportedFeatureIndex
		bitmask int64
		want    interface{}
		got     func(*currentstatus.PumpFeaturesV2Response) interface{}
	}{
		{"000012956576", 0, currentstatus.SupportedFeatureIndexMainFeatures, 1986368786, []currentstatus.PumpFeatureType{
			currentstatus.PumpFeatureDexcomG6Supported, currentstatus.PumpFeatureControlIQSupported, currentstatus.PumpFeatureWombatSupported,
			currentstatus.PumpFeatureBasalLimitSupported, currentstatus.PumpFeatureAutoPopSupported, currentstatus.PumpFeatureBLEPumpControlSupported,
			currentstatus.PumpFeaturePumpSettingsInIDPGUISupported,
		}, func(m *currentstatus.PumpFeaturesV2Response) interface{} { return m.PrimaryFeatures }},
		{"010100000000", 1, currentstatus.SupportedFeatureIndexControlIQProFeatures, 0, []currentstatus.ControlIQProFeatureType{},
			func(m *currentstatus.PumpFeaturesV2Response) interface{} { return m.ControlIQProFeatures }},
		{"0002ffeddd0e", 0, currentstatus.SupportedFeatureIndexControlFeatures, 249425407, []currentstatus.ControlFeatureType{
			currentstatus.ControlFeatureChangeCartridgeControl, currentstatus.ControlFeatureBasalDeliveryControl,
			currentstatus.ControlFeatureStandardBolusControl, currentstatus.ControlFeatureExtendedBolusControl,
			currentstatus.ControlFeatureTempRateControl, currentstatus.ControlFeatureCGMControl, currentstatus.ControlFeatureAAMControl,
			currentstatus.ControlFeatureShelfModeControl, currentstatus.ControlFeatureIDPControl, currentstatus.ControlFeatureUnregisterControl,
			currentstatus.ControlFeaturePumpTimeControl, currentstatus.ControlFeatureTZControl, currentstatus.ControlFeatureReminderSettingsControl,
			currentstatus.ControlFeatureAlertSettingsControl, currentstatus.ControlFeatureAnnumControl,
			currentstatus.ControlFeaturePumpGlobalSettingsControl, currentstatus.ControlFeatureFactoryConfigControl,
			currentstatus.ControlFeatureSnoozeSettingsControl, currentstatus.ControlFeatureFindMyPumpControl,
			currentstatus.ControlFeatureBolusOverflowControl, currentstatus.ControlFeatureUserInteractionControl,
			currentstatus.ControlFeatureBloodGlucoseEntryControl, currentstatus.ControlFeatureHLogCreatorControl,
		}, func(m *currentstatus.PumpFeaturesV2Response) interface{} { return m.ControlFeatures }},
		{"000380030000", 0, currentstatus.SupportedFeatureIndexControlIQFeatures, 896, []currentstatus.ControlIQFeatureType{
			currentstatus.ControlIQFeatureControlIQVersionOneFive, currentstatus.ControlIQFeatureTempRateEnabled,
		}, func(m *currentstatus.PumpFeaturesV2Response) interface{} { return m.ControlIQFeatures }},
		{"00047e000000", 0, currentstatus.SupportedFeatureIndexDexcomFeatures, 126, []currentstatus.DexcomFeatureType{
			currentstatus.DexcomFeatureG6CGM, currentstatus.DexcomFeatureG7CGM, currentstatus.DexcomFeatureAutoCal,
			currentstatus.DexcomFeatureDisableGraphSmoothing,
		}, func(m *currentstatus.PumpFeaturesV2Response) interface{} { return m.DexcomFeatures }},
		{"010500000000", 1, currentstatus.SupportedFeatureIndexAbbottFeatures, 0, []currentstatus.AbbottFeatureType{},
			func(m *currentstatus.PumpFeaturesV2Response) interface{} { return m.AbbottFeatures }},
		{"00062e000000", 0, currentstatus.SupportedFeatureIndexSecondaryFeatures, 46, []currentstatus.SecondaryFeatureType{
			currentstatus.SecondaryFeatureMultiCGM, currentstatus.SecondaryFeatureBLETDU,
		}, func(m *currentstatus.PumpFeaturesV2Response) interface{} { return m.SecondaryFeatures }},
	} {
		t.Run(tc.index.String(), func(t *testing.T) {
			expected := currentstatus.NewPumpFeaturesV2Response(tc.status, tc.index.ID(), tc.bitmask)
			parsed := parsePumpFeaturesV2Response(t, tc.cargo)
			if !bytes.Equal(parsed.Cargo(), expected.Cargo()) {
				t.Errorf("cargo %x, want %x", parsed.Cargo(), expected.Cargo())
			}
			if parsed.Status != tc.status || parsed.SupportedFeatureIndexID != tc.index.ID() || parsed.PumpFeaturesBitmask != tc.bitmask {
				t.Errorf("status %d, index %d, bitmask %d; want %d, %d, %d",
					parsed.Status, parsed.SupportedFeatureIndexID, parsed.PumpFeaturesBitmask, tc.status, tc.index.ID(), tc.bitmask)
			}
			if parsed.SupportedFeatureIndex() != tc.index {
				t.Errorf("index %v, want %v", parsed.SupportedFeatureIndex(), tc.index)
			}
			if got := tc.got(parsed); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("features %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPumpFeaturesV2ResponseSetsOnlyTheIndexedFeatures(t *testing.T) {
	m := parsePumpFeaturesV2Response(t, "000380030000")
	if m.PrimaryFeatures != nil || m.ControlFeatures != nil || m.DexcomFeatures != nil || m.ControlIQFeatures == nil {
		t.Errorf("features of other indexes set: %+v", m)
	}
}

func TestPumpFeaturesV2ResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "PumpFeaturesV2Response")
}
