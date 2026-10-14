package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestGlobalMaxBolusSettingsResponse(t *testing.T) {
	parity.TestMessage(t, "00188d1804a8611027e5ba", 24, 1, messages.CurrentStatus,
		currentstatus.NewGlobalMaxBolusSettingsResponse(25000, 10000))
}

func TestGlobalMaxBolusSettingsResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "GlobalMaxBolusSettingsResponse")
}
