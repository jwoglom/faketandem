package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetIDPSettingsRequest(t *testing.T) {
	for _, tc := range []struct {
		name            string
		insulinDuration int32
		changeType      control.ChangeType
		txID            int
		packets         []string
	}{
		{"profile 1 carb entry on", 240, control.ChangeTypeChangeCarbEntry, 9,
			[]string{"0109ac091e0101f00001043d493e20068b9bf3e9", "0009a18a33a4d6cb387d989b15ea9fceda7fc7"}},
		{"profile 1 insulin duration 4h", 240, control.ChangeTypeChangeInsulinDuration, -11,
			[]string{"01f5acf51e0101f000010122493e206e6dc237ab", "00f5dc6d11544e6f1fff07ae7674085f09651d"}},
		{"profile 1 insulin duration 5h", 300, control.ChangeTypeChangeInsulinDuration, 16,
			[]string{"0110ac101e01012c0101014f493e206cd2e4e729", "0010597f8b90affef6e13b0d1f36e671197e34"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := control.NewSetIDPSettingsRequest(1, 1, tc.insulinDuration, 1, tc.changeType)
			parity.TestMessage(t, tc.packets[0], tc.txID, 1, messages.Control, expected, tc.packets[1:]...)
		})
	}
}

func TestSetIDPSettingsRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetIDPSettingsRequest")
}
