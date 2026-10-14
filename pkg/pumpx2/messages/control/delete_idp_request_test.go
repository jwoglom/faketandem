package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestDeleteIDPRequest(t *testing.T) {
	for _, tc := range []struct {
		name    string
		idpID   int32
		txID    int
		packets []string
	}{
		{"profile 2", 2, 77, []string{"014dae4d1a0201c0493e208cb42b3ffcb6f3f0a9", "004de51d3ea637ac1b02df4cce79c8"}},
		{"profile 1", 1, 83, []string{"0153ae531a0101cf493e200e7a1458b74d619de6", "0053faa1644c35799cdc81ac28666a"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parity.TestMessage(t, tc.packets[0], tc.txID, 1, messages.Control, control.NewDeleteIDPRequest(tc.idpID, 1), tc.packets[1:]...)
		})
	}
}

func TestDeleteIDPRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "DeleteIDPRequest")
}
