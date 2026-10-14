package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestBasalLimitSettingsResponse(t *testing.T) {
	parity.TestMessage(t, "00038b030888130000b80b0000a097", 3, 1, messages.CurrentStatus,
		currentstatus.NewBasalLimitSettingsResponse(5000, 3000))
}

func TestBasalLimitSettingsResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "BasalLimitSettingsResponse")
}
