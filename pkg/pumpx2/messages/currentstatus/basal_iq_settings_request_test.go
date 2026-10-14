package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestBasalIQSettingsRequest(t *testing.T) {
	parity.TestMessage(t, "000462040053f5", 4, 1, messages.CurrentStatus, currentstatus.NewBasalIQSettingsRequest())
}

func TestBasalIQSettingsRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "BasalIQSettingsRequest")
}
