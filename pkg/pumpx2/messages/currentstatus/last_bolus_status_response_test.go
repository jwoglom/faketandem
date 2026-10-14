package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestLastBolusStatusResponse(t *testing.T) {
	expected := currentstatus.NewLastBolusStatusResponse(1, 10690, 463703230, 200, 3, 8, 8, 0, make([]byte, 2))
	parsed := parity.TestMessage(t, "000531051401c2290000be8ca31bc800000003080800000000bc1f", 5, 2, messages.CurrentStatus,
		expected).(*currentstatus.LastBolusStatusResponse)
	if s := parsed.BolusStatus(); s == nil || *s != currentstatus.BolusStatusComplete {
		t.Errorf("bolus status %v", s)
	}
}

func TestLastBolusStatusResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "LastBolusStatusResponse")
}
