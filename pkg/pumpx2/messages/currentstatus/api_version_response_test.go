package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestAPIVersionResponse(t *testing.T) {
	for _, tc := range []struct {
		name         string
		hex          string
		txID         int
		major, minor int32
	}{
		{"t:slim X2 7.3.1", "00022102040200000077c8", 2, 2, 0},
		{"t:slim X2 7.4", "00072107040200010047b8", 7, 2, 1},
		{"Mobi", "000421040403000500d7cc", 4, 3, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed := parity.TestMessage(t, tc.hex, tc.txID, 1, messages.CurrentStatus,
				currentstatus.NewAPIVersionResponse(tc.major, tc.minor)).(*currentstatus.APIVersionResponse)
			if parsed.MajorVersion != tc.major || parsed.MinorVersion != tc.minor {
				t.Errorf("version %d.%d, want %d.%d", parsed.MajorVersion, parsed.MinorVersion, tc.major, tc.minor)
			}
		})
	}
}

func TestAPIVersionResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ApiVersionResponse")
}
