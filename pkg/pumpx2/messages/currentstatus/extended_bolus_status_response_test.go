package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestExtendedBolusStatusResponseWithNoExtendedBolus(t *testing.T) {
	parity.TestMessage(t, "00062f06120000000000000000000000000000000000007cdd", 6, 2, messages.CurrentStatus,
		currentstatus.NewExtendedBolusStatusResponse(0, 0, 0, 0, 0, 0))
}

func TestExtendedBolusStatusResponseReadsTheTimestampFromWhereItIsNotWritten(t *testing.T) {
	built := currentstatus.NewExtendedBolusStatusResponse(1, 10, 0x01020304, 0, 0, 0)
	parsed := &currentstatus.ExtendedBolusStatusResponse{}
	if err := parsed.Parse(built.Cargo()); err != nil {
		t.Fatal(err)
	}
	if parsed.BolusID != 10 || parsed.Timestamp != 0x0102 {
		t.Errorf("bolus %d at %#x, want bolus 10 at 0x102", parsed.BolusID, parsed.Timestamp)
	}
}

func TestExtendedBolusStatusResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ExtendedBolusStatusResponse")
}
