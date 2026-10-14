package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestChangeControlIQSettingsResponse(t *testing.T) {
	expected, err := control.NewChangeControlIQSettingsResponseFromCargo(make([]byte, 3))
	if err != nil {
		t.Fatal(err)
	}
	parity.TestMessage(t, "005ccb5c1b000000d9a3ee1f799d9f28e8a6759fed2cb0fafeffdea01a2b0cf56bb0", 92, 1, messages.Control, expected)
}

func TestChangeControlIQSettingsResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ChangeControlIQSettingsResponse")
}
