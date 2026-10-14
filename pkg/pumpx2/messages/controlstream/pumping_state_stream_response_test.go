package controlstream_test

import (
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/controlstream"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestPumpingStateStreamResponseParsesAsExitFillTubingModeWithItsStates(t *testing.T) {
	states := []controlstream.PumpingState{
		controlstream.PumpingStateIsDeliveringTherapy, controlstream.PumpingStateBolusAllowed,
		controlstream.PumpingStateDeliveringBasal, controlstream.PumpingStateCartridgeIsInstalled,
	}
	built := controlstream.NewPumpingStateStreamResponse(true, controlstream.PumpingStateToBitmask(states...))
	parsed := &controlstream.ExitFillTubingModeStateStreamResponse{}
	if err := parsed.Parse(built.Cargo()); err != nil {
		t.Fatal(err)
	}
	if !parsed.IsPumpingStateSetAfterStartUp || !reflect.DeepEqual(parsed.States(), states) {
		t.Errorf("parsed %+v, states %v, want %v", parsed, parsed.States(), states)
	}
}

func TestPumpingStateStreamResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "PumpingStateStreamResponse")
}
