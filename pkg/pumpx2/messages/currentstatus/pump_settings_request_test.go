package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestPumpSettingsRequest(t *testing.T) {
	parity.TestMessage(t, "0004520400f630", 4, 1, messages.CurrentStatus, currentstatus.NewPumpSettingsRequest())
}

func TestPumpSettingsRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "PumpSettingsRequest")
}
