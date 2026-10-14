package currentstatus_test

import (
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestIDPSegmentResponse(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		hex                   string
		txID                  int
		segment, start, basal int32
	}{
		{"midnight", "000443040f0000000020037017000078001e000f3078", 4, 0, 0, 800},
		{"6am", "000543050f00016801e2047017000078001e000f121d", 5, 1, 360, 1250},
		{"11am", "000643060f00029402e8037017000078001e000f9460", 6, 2, 660, 1000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := currentstatus.NewIDPSegmentResponse(0, tc.segment, tc.start, tc.basal, 6000, 120, 30, 15)
			parsed := parity.TestMessage(t, tc.hex, tc.txID, 2, messages.CurrentStatus, expected).(*currentstatus.IDPSegmentResponse)
			want := []currentstatus.IDPSegmentStatus{
				currentstatus.IDPSegmentStatusBasalRate, currentstatus.IDPSegmentStatusCarbRatio,
				currentstatus.IDPSegmentStatusTargetBG, currentstatus.IDPSegmentStatusCorrectionFactor,
			}
			if !reflect.DeepEqual(parsed.IDPStatus, want) {
				t.Errorf("status %v, want %v", parsed.IDPStatus, want)
			}
		})
	}
}

func TestIDPSegmentResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "IDPSegmentResponse")
}
