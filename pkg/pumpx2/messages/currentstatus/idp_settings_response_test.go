package currentstatus_test

import (
	"strings"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestIDPSettingsResponse(t *testing.T) {
	for _, tc := range []struct {
		name                                string
		hex                                 string
		txID                                int
		idpID                               int32
		profile                             string
		segments, insulinDuration, maxBolus int32
	}{
		{"one profile", "00044104170041000000000000000000000000000000042c01a86101c2a5", 4, 0, "A", 4, 300, 25000},
		{"8 hours", "0005410517014e61000000000000000000000000000001e001a861015ef2", 5, 1, "Na", 1, 480, 25000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := currentstatus.NewIDPSettingsResponse(tc.idpID, tc.profile, tc.segments, tc.insulinDuration, tc.maxBolus, true)
			if err != nil {
				t.Fatal(err)
			}
			parity.TestMessage(t, tc.hex, tc.txID, 2, messages.CurrentStatus, expected)
		})
	}
}

func TestIDPSettingsResponseRefusesANameOver16Bytes(t *testing.T) {
	if _, err := currentstatus.NewIDPSettingsResponse(0, strings.Repeat("é", 9), 1, 300, 25000, true); err == nil {
		t.Error("an 18-byte name was built")
	}
}

func TestIDPSettingsResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "IDPSettingsResponse")
}
