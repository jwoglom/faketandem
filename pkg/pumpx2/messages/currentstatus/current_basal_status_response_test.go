package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCurrentBasalStatusResponse(t *testing.T) {
	parsed := parity.TestMessage(t, "0003290309200300000000000001d8d0", 3, 1, messages.CurrentStatus,
		currentstatus.NewCurrentBasalStatusResponse(800, 0, 1)).(*currentstatus.CurrentBasalStatusResponse)
	if parsed.ProfileBasalRate != 800 || parsed.CurrentBasalRate != 0 || parsed.BasalModifiedBitmask != 1 {
		t.Errorf("parsed %+v", parsed)
	}
}

func TestCurrentBasalStatusResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CurrentBasalStatusResponse")
}
