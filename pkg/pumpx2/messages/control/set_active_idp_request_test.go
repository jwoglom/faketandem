package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetActiveIDPRequest(t *testing.T) {
	for _, tc := range []struct {
		name    string
		cargo   []byte
		txID    int
		packets []string
	}{
		{"profile with ID 1", []byte{1, 1}, 25, []string{"0119ec191a010165493e2009a33736acb3713492", "00199d7598653853bd78abd443ec3b"}},
		{"profile with ID 0", []byte{0, 1}, 36, []string{"0124ec241a00017c493e2027b0811608d99b2431", "00249dec87124ebe4216766ea86133"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := control.NewSetActiveIDPRequestFromCargo(tc.cargo)
			if err != nil {
				t.Fatal(err)
			}
			parity.TestMessage(t, tc.packets[0], tc.txID, 1, messages.Control, expected, tc.packets[1:]...)
		})
	}
}

func TestSetActiveIDPRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetActiveIDPRequest")
}
