package currentstatus_test

import (
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCurrentBolusStatusResponse(t *testing.T) {
	parsed := parity.TestMessage(t, "00032d030f000000000000000000000000000000b109", 3, 2, messages.CurrentStatus,
		currentstatus.NewCurrentBolusStatusResponse(0, 0, 0, 0, 0, 0)).(*currentstatus.CurrentBolusStatusResponse)
	if parsed.Valid() {
		t.Error("an empty bolus status is valid")
	}
	if s := parsed.Status(); s == nil || *s != currentstatus.CurrentBolusStatusAlreadyDeliveredOrInvalid {
		t.Errorf("status %v, want ALREADY_DELIVERED_OR_INVALID", s)
	}
}

func TestCurrentBolusStatusResponseDelivering(t *testing.T) {
	m := currentstatus.NewCurrentBolusStatusResponse(currentstatus.CurrentBolusStatusDelivering.ID(), 10650, 461510642, 1000, 0, 8)
	parsed := &currentstatus.CurrentBolusStatusResponse{}
	if err := parsed.Parse(m.Cargo()); err != nil {
		t.Fatal(err)
	}
	if messages.ToString(parsed) != messages.ToString(m) {
		t.Errorf("parsed %s, want %s", messages.ToString(parsed), messages.ToString(m))
	}
	if !parsed.Valid() {
		t.Error("a delivering bolus is not valid")
	}
	if want := time.Date(2022, 8, 16, 13, 24, 2, 0, time.UTC); !parsed.Time().Equal(want) {
		t.Errorf("time %s, want %s", parsed.Time(), want)
	}
}

func TestCurrentBolusStatusResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CurrentBolusStatusResponse")
}
