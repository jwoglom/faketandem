package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCGMRateAlertSettingsResponse(t *testing.T) {
	parity.TestMessage(t, "00045d04060301010301016b8c", 4, 1, messages.CurrentStatus,
		currentstatus.NewCGMRateAlertSettingsResponse(3, 1, 1, 3, 1, 1))
	parity.TestMessage(t, "00055d0506030101020100a9ec", 5, 1, messages.CurrentStatus,
		currentstatus.NewCGMRateAlertSettingsResponse(3, 1, 1, 2, 1, 0))
}

func TestCGMRateAlertSettingsResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CGMRateAlertSettingsResponse")
}
