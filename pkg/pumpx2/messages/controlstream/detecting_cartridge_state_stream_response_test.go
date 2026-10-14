package controlstream_test

import (
	"fmt"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/controlstream"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

// ignoringHmac is pumpX2's initPumpState(IGNORE_INVALID_HMAC, 0): the
// captures' signatures are under a pairing code the tests do not have.
var ignoringHmac = parity.MessageTester{Env: messages.Env{PairingCode: "IGNORE_HMAC_SIGNATURE_EXCEPTION", TimeSinceReset: "0"}}

func TestDetectingCartridgeStateStreamResponse(t *testing.T) {
	for _, tc := range []struct {
		hex     string
		percent byte
	}{
		{"0089e3891a140078c84020f4eabab90ec17db0da1b4d99095f92c063a1b1b9456a", 20},
		{"0089e3891a280078c840207ae3354ae4a861ca6cbaf2a6acc178441f68714f8a3f", 40},
		{"0089e3891a3c0078c840204b662d159bd8d8827045926b2b16c18dc867fa8b2bdd", 60},
		{"0089e3891a500078c840209382ea406c228d99294fb75a06b6034720bebf558af6", 80},
		{"0089e3891a640078c840200446e27abb9bac31f8d1183f53dae99a2cc61d49b7ba", 100},
	} {
		t.Run(fmt.Sprintf("%d%%", tc.percent), func(t *testing.T) {
			expected, err := controlstream.NewDetectingCartridgeStateStreamResponseFromCargo([]byte{tc.percent, 0})
			if err != nil {
				t.Fatal(err)
			}
			parsed := ignoringHmac.Test(t, tc.hex, 0x89, 1, messages.ControlStream, expected).(*controlstream.DetectingCartridgeStateStreamResponse)
			if parsed.PercentComplete != int32(tc.percent) || parsed.LoadCartridgeState || parsed.IsComplete() != (tc.percent == 100) {
				t.Errorf("parsed %+v", parsed)
			}
		})
	}
}

func TestDetectingCartridgeStateStreamResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "DetectingCartridgeStateStreamResponse")
}
