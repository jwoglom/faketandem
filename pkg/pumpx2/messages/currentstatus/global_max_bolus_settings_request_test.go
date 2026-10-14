package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestGlobalMaxBolusSettingsRequest(t *testing.T) {
	parity.TestMessage(t, "00038c0300f4d7", 3, 1, messages.CurrentStatus, currentstatus.NewGlobalMaxBolusSettingsRequest())
}

func TestGlobalMaxBolusSettingsRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "GlobalMaxBolusSettingsRequest")
}
