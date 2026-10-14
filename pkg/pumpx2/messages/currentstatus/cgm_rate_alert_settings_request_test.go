package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCGMRateAlertSettingsRequest(t *testing.T) {
	parity.TestMessage(t, "00045c0400f72b", 4, 1, messages.CurrentStatus, currentstatus.NewCGMRateAlertSettingsRequest())
}

func TestCGMRateAlertSettingsRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CGMRateAlertSettingsRequest")
}
