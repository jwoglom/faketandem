package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCGMHardwareInfoResponse(t *testing.T) {
	expected, err := currentstatus.NewCGMHardwareInfoResponse("8RR239", 0)
	if err != nil {
		t.Fatal(err)
	}
	parsed := parity.TestMessage(t, "00036103113852523233390000000000000000000000a2a3", 3, 2, messages.CurrentStatus,
		expected).(*currentstatus.CGMHardwareInfoResponse)
	if parsed.HardwareInfoString != "8RR239" || parsed.LastByte != 0 {
		t.Errorf("parsed %+v", parsed)
	}
}

func TestCGMHardwareInfoResponseRefusesALongID(t *testing.T) {
	if _, err := currentstatus.NewCGMHardwareInfoResponse("0123456789abcdefg", 0); err == nil {
		t.Error("a 17-byte ID fits")
	}
}

func TestCGMHardwareInfoResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CGMHardwareInfoResponse")
}
