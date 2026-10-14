package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestPumpGlobalsResponse(t *testing.T) {
	vibrate, low := currentstatus.AnnunciationVibrate.ID(), currentstatus.AnnunciationAudioLow.ID()
	expected := currentstatus.NewPumpGlobalsResponse(0, 500, 2000, 0, 0, vibrate, low, low, vibrate, vibrate, low, low)
	parity.TestMessage(t, "000357030e00f401d007000003020203030202050f", 3, 2, messages.CurrentStatus, expected)
}

func TestPumpGlobalsResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "PumpGlobalsResponse")
}
