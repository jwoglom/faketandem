package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestIDPSegmentRequest(t *testing.T) {
	parity.TestMessage(t, "0004420402000076e0", 4, 1, messages.CurrentStatus, currentstatus.NewIDPSegmentRequest(0, 0))
}

func TestIDPSegmentRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "IDPSegmentRequest")
}
