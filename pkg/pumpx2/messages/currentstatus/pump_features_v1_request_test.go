package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestPumpFeaturesV1Request(t *testing.T) {
	parity.TestMessage(t, "00034e0300639f", 3, 1, messages.CurrentStatus, currentstatus.NewPumpFeaturesV1Request())
}

func TestPumpFeaturesV1RequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "PumpFeaturesV1Request")
}
