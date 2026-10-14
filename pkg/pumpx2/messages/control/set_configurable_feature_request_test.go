package control_test

import (
	"encoding/hex"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetConfigurableFeatureRequest(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		groupIndex, featureIndex int32
		featureState             bool
		cargo                    string
	}{
		{"enable", 2, 5, true, "020501"},
		{"disable", 0, 0, false, "000000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := control.NewSetConfigurableFeatureRequest(tc.groupIndex, tc.featureIndex, tc.featureState)
			parsed := &control.SetConfigurableFeatureRequest{}
			if err := parsed.Parse(expected.Cargo()); err != nil {
				t.Fatal(err)
			}
			if got := hex.EncodeToString(parsed.Cargo()); got != tc.cargo {
				t.Errorf("cargo %s, want %s", got, tc.cargo)
			}
			if parsed.GroupIndex != tc.groupIndex || parsed.FeatureIndex != tc.featureIndex || parsed.FeatureState != tc.featureState {
				t.Errorf("parsed %+v", parsed)
			}
		})
	}
}

func TestSetConfigurableFeatureRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetConfigurableFeatureRequest")
}
