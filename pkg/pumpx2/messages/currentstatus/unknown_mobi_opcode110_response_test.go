package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestUnknownMobiOpcode110Response(t *testing.T) {
	for _, tc := range []struct {
		hex   string
		txID  int
		cargo []byte
	}{
		{"00e36fe304d1c00000a3fc", -29, []byte{0xd1, 0xc0, 0, 0}},
		{"00e46fe404d1c00000e234", -28, []byte{0xd1, 0xc0, 0, 0}},
		{"00fa6ffa0439c200008cc1", -6, []byte{0x39, 0xc2, 0, 0}},
	} {
		expected, err := currentstatus.NewUnknownMobiOpcode110ResponseFromCargo(tc.cargo)
		if err != nil {
			t.Fatal(err)
		}
		parity.TestMessage(t, tc.hex, tc.txID, 1, messages.CurrentStatus, expected)
	}
}

func TestUnknownMobiOpcode110ResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "UnknownMobiOpcode110Response")
}
