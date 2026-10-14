package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestPumpFeaturesV2Request(t *testing.T) {
	parity.TestMessage(t, "0006a006010027ef", 6, 1, messages.CurrentStatus, currentstatus.NewPumpFeaturesV2RequestFromID(0))
	parity.TestMessage(t, "0004a00401016691", 4, 1, messages.CurrentStatus, currentstatus.NewPumpFeaturesV2RequestFromID(1))
	parity.TestMessage(t, "0004a00401016691", 4, 1, messages.CurrentStatus,
		currentstatus.NewPumpFeaturesV2Request(currentstatus.SupportedFeatureIndexControlIQProFeatures))
}

func TestPumpFeaturesV2RequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "PumpFeaturesV2Request")
}
