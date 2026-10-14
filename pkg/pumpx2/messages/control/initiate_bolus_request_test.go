package control_test

import (
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

// Captured with the pairing code and time since reset each was signed with;
// the cargo is the request's first 37 bytes, without its signature.
func TestInitiateBolusRequest(t *testing.T) {
	for _, tc := range []struct {
		name           string
		pairingCode    string
		timeSinceReset string
		cargo          string
		txID           int
		packets        []string
		want           control.InitiateBolusRequest
	}{
		{
			name: "1u correction", pairingCode: "6VeDeRAL5DCigGw2", timeSinceReset: "461510642", txID: 57,
			cargo: "e80300009a2900000800000000000000000000000000000000000000000000000000000000",
			packets: []string{"03399e393de80300009a29000008000000000000", "0239000000000000000000000000000000000000",
				"013900000000f217821b68f94cebe6717a5d1551", "003927147fec9ad979926aaccd74"},
			want: control.InitiateBolusRequest{TotalVolume: 1000, BolusID: 10650, BolusTypeBitmask: 8},
		},
		{
			name: "0.13u for 13g at 142 mg/dL", pairingCode: "6VeDeRAL5DCigGw2", timeSinceReset: "461589180", txID: 62,
			cargo: "820000009c2900000182000000000000000d008e0000000000000000000000000000000000",
			packets: []string{"033e9e3e3d820000009c29000001820000000000", "023e00000d008e00000000000000000000000000",
				"013e00000000bc4a831b9cbf19ffb856288a8afa", "003e8f24a463e00cf3bbe5d305dd"},
			want: control.InitiateBolusRequest{TotalVolume: 130, BolusID: 10652, BolusTypeBitmask: 1, FoodVolume: 130, BolusCarbs: 13, BolusBG: 142},
		},
		{
			name: "0.11u for 11g at 161 mg/dL with 0.13u IOB", pairingCode: "6VeDeRAL5DCigGw2", timeSinceReset: "461589420", txID: 57,
			cargo: "6e0000009d290000016e000000000000000b00a10082000000000000000000000000000000",
			packets: []string{"03399e393d6e0000009d290000016e0000000000", "023900000b00a100820000000000000000000000",
				"013900000000ac4b831b7a0b7cfc14a30b9c3995", "0039dc8bbbdfa2ce2ce995725407"},
			want: control.InitiateBolusRequest{TotalVolume: 110, BolusID: 10653, BolusTypeBitmask: 1, FoodVolume: 110, BolusCarbs: 11, BolusBG: 161, BolusIOB: 130},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cargo, _ := hex.DecodeString(tc.cargo)
			expected, err := control.NewInitiateBolusRequestFromCargo(cargo)
			if err != nil {
				t.Fatal(err)
			}
			tester := parity.MessageTester{Env: messages.Env{PairingCode: tc.pairingCode, TimeSinceReset: tc.timeSinceReset}}
			parsed := tester.Test(t, tc.packets[0], tc.txID, 3, messages.Control, expected, tc.packets[1:]...).(*control.InitiateBolusRequest)
			tc.want.SetCargo(parsed.Cargo())
			if !reflect.DeepEqual(*parsed, tc.want) {
				t.Errorf("parsed %+v, want %+v", *parsed, tc.want)
			}
		})
	}
}

func TestInitiateBolusRequestRefusesABolusBelowTheMinimum(t *testing.T) {
	if _, err := control.NewInitiateBolusRequest(control.MinBolusMilliunits-1, 1, 1, 0, 0, 0, 0, 0); err == nil {
		t.Error("a bolus below the minimum was built")
	}
	if _, err := control.NewExtendedInitiateBolusRequest(0, 1, 4, 0, 0, 0, 0, 0, control.MinExtendedBolusMilliunits, 3600, 0); err != nil {
		t.Errorf("an extended bolus at the minimum: %v", err)
	}
}

func TestInitiateBolusRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "InitiateBolusRequest")
}
