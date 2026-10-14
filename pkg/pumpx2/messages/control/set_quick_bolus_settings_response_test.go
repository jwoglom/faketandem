package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetQuickBolusSettingsResponse(t *testing.T) {
	expected, err := control.NewSetQuickBolusSettingsResponseFromCargo([]byte{0})
	if err != nil {
		t.Fatal(err)
	}
	parsed := parity.TestMessage(t, "004ed34e19002a6ff01fd8f817c3af77277060ce0451da1741ef3caa98d25b8a", 78, 1, messages.Control,
		expected).(*control.SetQuickBolusSettingsResponse)
	if parsed.Status != 0 {
		t.Errorf("status %d", parsed.Status)
	}
}

func TestSetQuickBolusSettingsResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetQuickBolusSettingsResponse")
}
