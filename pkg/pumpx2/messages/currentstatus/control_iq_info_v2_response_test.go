package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestControlIQInfoV2Response(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		hex                       string
		txID                      int
		mode, byte6, byte7, byte8 int32
	}{
		{"API 2.5", "000cb30c13019600014b01000101030000000000000000002d45", 12, 1, 0, 1, 1},
		{"standard", "005ab35a13019600014b000102000300000000000000000034ef", 90, 0, 1, 2, 0},
		{"sleep", "0024b32413019600014b0100010003000000000000000000a95f", 36, 1, 0, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parity.TestMessage(t, tc.hex, tc.txID, 2, messages.CurrentStatus,
				currentstatus.NewControlIQInfoV2Response(true, 150, 1, 75, tc.mode, tc.byte6, tc.byte7, tc.byte8, 3, 0, 0, 0))
		})
	}
}

func TestControlIQInfoV2ResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ControlIQInfoV2Response")
}
