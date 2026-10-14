package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCGMOORAlertSettingsRequest(t *testing.T) {
	parity.TestMessage(t, "00035e030000dc", 3, 1, messages.CurrentStatus, currentstatus.NewCGMOORAlertSettingsRequest())
}

func TestCGMOORAlertSettingsRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CGMOORAlertSettingsRequest")
}
