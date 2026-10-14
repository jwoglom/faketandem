package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestBasalLimitSettingsRequest(t *testing.T) {
	parity.TestMessage(t, "00048a0400c3fc", 4, 1, messages.CurrentStatus, currentstatus.NewBasalLimitSettingsRequest())
}

func TestBasalLimitSettingsRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "BasalLimitSettingsRequest")
}
