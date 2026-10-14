package controlstream_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/controlstream"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestFillTubingStateStreamResponse(t *testing.T) {
	for _, tc := range []struct {
		name   string
		hex    string
		txID   byte
		button byte
	}{
		{"1 down", "0076e5761901dc9bee1fc0a8edcd9cbee43c109b046853fab2051262f1b84ca9", 0x76, 1},
		{"2 up", "0076e5761900e29bee1f69b046e6713fd3947e435f7a10381e7864ac91091a79", 0x76, 0},
		{"3 down", "0076e5761901e29bee1fe0744f64ededbc751612b0e00f367de5eaeccf5bfc96", 0x76, 1},
		{"4 up", "0076e5761900e39bee1f9d9e1e59f25f697025e559a930e69f91498ace97b83f", 0x76, 0},
		{"5 down", "0076e5761901e69bee1ffab75ebc85b1a37c7309d479a4038afb7e4d6fdb91c7", 0x76, 1},
		{"6 up", "0076e5761900ed9bee1f2a9429c51924b746e4a16d31d2981ce4e00e06b75eff", 0x76, 0},
		{"7 down", "0076e5761901f49bee1ff790d0d466213d4b56db4072b38c576a9a575be6b650", 0x76, 1},
		{"8 up", "0076e5761900f59bee1f9f053315303c66641ccc9e34555b50e153b58f8bc523", 0x76, 0},
		{"B 9 up", "0089e5891900249cee1f9b9e05be753ab197b4036913314c1266af76a668266a", 0x89, 0},
		{"C 10 down", "0092e5921901379cee1fc0b37184a090390ff6bd41b1ac5566a0d66a53d6f74e", 0x92, 1},
		{"C 11 up", "0092e5921900449cee1f7714edde7740ac55f395f7c3030daf9da601fb86d7ae", 0x92, 0},
		{"D 12 up", "009de59d1900639cee1f835f372ea7bf7587005915997b64cc35afd9f06d9b1b", 0x9d, 0},
		{"E 13 down", "00a5e5a51901689cee1f3def52456aa152c855a664ea2f1d7846de2699807f3b", 0xa5, 1},
		{"E 14 up", "00a5e5a51900769cee1fe8533421e37d68f6ee8b2b226361aa9282fe729065a5", 0xa5, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := controlstream.NewFillTubingStateStreamResponseFromCargo([]byte{tc.button})
			if err != nil {
				t.Fatal(err)
			}
			parsed := testCapture(t, tc.hex, tc.txID, expected).(*controlstream.FillTubingStateStreamResponse)
			if parsed.ButtonDown() != (tc.button == 1) {
				t.Errorf("button state %d", parsed.ButtonState)
			}
		})
	}
}

func TestFillTubingStateStreamResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "FillTubingStateStreamResponse")
}
