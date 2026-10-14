package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetIDPSettingsResponse(t *testing.T) {
	for _, tc := range []struct {
		name string
		hex  string
		txID int
	}{
		{"captured response 1", "0009ad091a00020e033e2061ea619a9325f6bd309c77dc8c496c9c352a4da72237", 9},
		{"captured response 2", "0000ad001a000200033e208cb5056600c4b9485675759170d5b20c687e24c64bea", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parity.TestMessage(t, tc.hex, tc.txID, 1, messages.Control, control.NewSetIDPSettingsResponse(0))
		})
	}
}

func TestSetIDPSettingsResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetIDPSettingsResponse")
}
