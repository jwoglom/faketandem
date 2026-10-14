package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCGMOORAlertSettingsResponse(t *testing.T) {
	parity.TestMessage(t, "00035f0303140105be32", 3, 1, messages.CurrentStatus,
		currentstatus.NewCGMOORAlertSettingsResponse(20, 1, 5))
}

func TestCGMOORAlertSettingsResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CGMOORAlertSettingsResponse")
}
