package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestLocalizationResponse(t *testing.T) {
	for _, tc := range []struct {
		name      string
		hex       string
		txID      int
		languages int64
	}{
		{"t:slim X2", "0004a70407000002010000003a49", 4, 1},
		{"Mobi", "0035a7350700000200000000af20", 53, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed := parity.TestMessage(t, tc.hex, tc.txID, 1, messages.CurrentStatus,
				currentstatus.NewLocalizationResponse(0, 0, 2, tc.languages)).(*currentstatus.LocalizationResponse)
			if parsed.GlucoseUOM != 0 || parsed.LanguageSelected != 0 || parsed.RegionSetting != 2 || parsed.LanguagesAvailableBitmask != tc.languages {
				t.Errorf("parsed %+v", parsed)
			}
		})
	}
}

func TestLocalizationResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "LocalizationResponse")
}
