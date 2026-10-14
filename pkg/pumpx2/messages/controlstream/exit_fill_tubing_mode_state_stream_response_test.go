package controlstream_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/controlstream"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestExitFillTubingModeStateStreamResponse(t *testing.T) {
	expected := controlstream.NewExitFillTubingModeStateStreamResponse(0)
	parsed := testCapture(t, "0091e9911900aec84020f5be25220634beb8ffff2cad2cf936582ee0226d2f28", 0x91,
		expected).(*controlstream.ExitFillTubingModeStateStreamResponse)
	if parsed.State == nil || *parsed.State != controlstream.ExitFillTubingModeStateNotComplete {
		t.Errorf("state %v, want NOT_COMPLETE", parsed.State)
	}
}

func TestExitFillTubingModeStateStreamResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ExitFillTubingModeStateStreamResponse")
}
