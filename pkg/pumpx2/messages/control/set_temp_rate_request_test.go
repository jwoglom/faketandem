package control_test

import (
	"encoding/hex"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetTempRateRequest(t *testing.T) {
	for _, tc := range []struct {
		name             string
		minutes, percent int32
		// fromCargo builds the expected request from cargo rather than from
		// minutes and percent, as pumpX2's test does for these captures.
		fromCargo      bool
		cargo          string
		timeSinceReset string
		packets        []string
		txID           int
	}{
		{"0% for 15m", 15, 0, false, "a0bb0d000000", "", []string{"0151a4511ea0bb0d0000005f6ef01fd2ed2a3276", "005179ee6512fde74533e1f1b7d863347d27bf"}, 81},
		{"1% for 15m", 15, 1, false, "a0bb0d000100", "", []string{"017da47d1ea0bb0d000100936ef01fa736f5a248", "007d1af9111b009e52e04b30d22241635de6bf"}, 125},
		{"2% for 15m", 15, 2, false, "a0bb0d000200", "", []string{"0190a4901ea0bb0d000200ab6ef01f8add3e9729", "0090fde703c39a6278e3f2fd2a061b9c8dd8ee"}, -112},
		{"10% for 15m", 15, 10, false, "a0bb0d000a00", "", []string{"01a2a4a21ea0bb0d000a00bc6ef01f0c373adbe4", "00a273ec2f89dc1ce285547b87c0bb95138a64"}, -94},
		{"40% for 15m", 15, 40, false, "a0bb0d002800", "", []string{"01b5a4b51ea0bb0d002800d46ef01fc26d64aeb1", "00b5c5a1530bf58d12e57e4f1bf1699c8f3adf"}, -75},
		{"50% for 15m", 15, 50, false, "a0bb0d003200", "", []string{"01c8a4c81ea0bb0d003200e66ef01fcea48f2fac", "00c826336f9f6a8f34304818abf9e13ca37408"}, -56},
		{"51% for 15m", 15, 51, false, "a0bb0d003300", "", []string{"01d8a4d81ea0bb0d003300fd6ef01f1ad8f7953c", "00d852535070f4e7bfeb0b02bb64144c15b6f4"}, -40},
		{"52% for 15m", 15, 52, false, "a0bb0d003400", "", []string{"01e7a4e71ea0bb0d0034000a6ff01f3d7fba8667", "00e7cc160c89eb7201e359591d165fcdf8bebe"}, -25},
		{"98% for 15m", 15, 98, false, "a0bb0d006200", "", []string{"01fba4fb1ea0bb0d0062001f6ff01f7d67b16536", "00fbea93dfed8f17eb822192e68b924fd723df"}, -5},
		{"140% for 15m", 15, 140, false, "a0bb0d008c00", "", []string{"010da40d1ea0bb0d008c00316ff01f5a4be23492", "000dfab7fdac6cfc377fcbe1d3aa7d6bc6bb05"}, 13},
		{"180% for 15m", 15, 180, false, "a0bb0d00b400", "", []string{"011da41d1ea0bb0d00b400406ff01f9d6f9efde6", "001d9f6a71538a5b2a3add07b39a91fa759fba"}, 29},
		{"199% for 15m", 15, 199, false, "a0bb0d00c700", "", []string{"0131a4311ea0bb0d00c700526ff01febde639dbf", "003157339f103c7394932d70ed467524abc61d"}, 49},
		{"201% for 15m", 15, 201, false, "a0bb0d00c900", "", []string{"0143a4431ea0bb0d00c900676ff01f7ebca592bc", "0043916ac521eded9e84d122e7b75e7a054586"}, 67},
		{"245% for 15m", 15, 245, false, "a0bb0d00f500", "", []string{"0155a4551ea0bb0d00f5007c6ff01f24b05dac9b", "0055e618d35684b87d8cce8c356302839098ae"}, 85},
		{"250% for 15m", 15, 250, false, "a0bb0d00fa00", "", []string{"0167a4671ea0bb0d00fa00896ff01fc697a2ed82", "00674d86c93d7fcc55a1a57c9a6e6b7c734b6c"}, 103},
		{"95% for 15m", 15, 95, false, "a0bb0d005f00", "1906112", []string{"0122a4221ea0bb0d005f001c998b1e37eacb5339", "00227c57675fc47f8c9bf5dec1f0d31d42a4b7"}, 34},
		{"0% for 16m", 16, 0, true, "00a60e000000", "", []string{"017aa47a1e00a60e0000009c6ff01fd518027a01", "007a9e3f8b12952dd41b013580028bd358981a"}, 122},
		{"0% for 17m", 17, 0, true, "60900f000000", "", []string{"018aa48a1e60900f000000ae6ff01fe6475f9d9b", "008a1c9e1920361d6075162964e5bee8b87eac"}, -118},
		{"0% for 18m", 18, 0, true, "c07a10000000", "", []string{"019ca49c1ec07a10000000c46ff01faa229302d8", "009cc3413fc22ebcaaa8b6919ca85c526bddca"}, -100},
		{"95% for 59m", 59, 95, true, "200436005f00", "", []string{"01afa4af1e200436005f00df6ff01f128b115990", "00afd1c77c1e849f7852275c0a1bc29c8d1956"}, -81},
		{"95% for 2h15m", 135, 95, true, "a0987b005f00", "", []string{"01bea4be1ea0987b005f00ec6ff01fdda86077e2", "00be42130e21edebfb4193239c2a4f18b9de14"}, -66},
		{"95% for 3h15m", 195, 95, true, "2087b2005f00", "", []string{"01d0a4d01e2087b2005f00fa6ff01fc5457a0f5f", "00d02c5dff03f795611fa4d724e0c2e2f081ee"}, -48},
		{"95% for 12h15m", 735, 95, true, "a0e9a0025f00", "", []string{"01e1a4e11ea0e9a0025f000e70f01fe0c7050329", "00e1c2b68ba9ab21c5b2890fbdff983c65faa6"}, -31},
		{"95% for 71h15m", 4275, 95, true, "20e1490f5f00", "", []string{"01f3a4f31e20e1490f5f002470f01f8da7ffa546", "00f33a8a503793fd1edeef7c7c63715a2e93d3"}, -13},
		{"95% for 71h15m again", 4275, 95, true, "20e1490f5f00", "", []string{"0103a4031e20e1490f5f003670f01f44d97042ee", "000306c26ba1833dd91cf172f3c2cb1d062486"}, 3},
		{"95% for 72h", 4320, 95, true, "0014730f5f00", "", []string{"0118a4181e0014730f5f004a70f01f79ae754dab", "00180002ed28543998f800552cd97030d29966"}, 24},
		{"101% for 15m", 15, 101, true, "a0bb0d006500", "", []string{"0148a4481ea0bb0d00650016eaee1fc5720c4461", "0048b8e23a6c4fb71331d0cbf3547996eb6cb5"}, 72},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cargo, _ := hex.DecodeString(tc.cargo)
			var expected *control.SetTempRateRequest
			var err error
			if tc.fromCargo {
				expected, err = control.NewSetTempRateRequestFromCargo(cargo)
			} else {
				expected, err = control.NewSetTempRateRequest(tc.minutes, tc.percent)
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := hex.EncodeToString(expected.Cargo()); got != tc.cargo {
				t.Errorf("cargo %s, want %s", got, tc.cargo)
			}
			if expected.Minutes != tc.minutes || expected.Percent != tc.percent {
				t.Errorf("%d%% for %d minutes, want %d%% for %d", expected.Percent, expected.Minutes, tc.percent, tc.minutes)
			}
			tester := parity.MessageTester{Env: messages.Env{TimeSinceReset: tc.timeSinceReset}}
			tester.Test(t, tc.packets[0], tc.txID, 1, messages.Control, expected, tc.packets[1:]...)
		})
	}
}

func TestSetTempRateRequestRefusesARateThePumpDoesNotSupport(t *testing.T) {
	for _, tc := range []struct{ minutes, percent int32 }{{14, 100}, {4321, 100}, {30, -1}, {30, 251}} {
		if _, err := control.NewSetTempRateRequest(tc.minutes, tc.percent); err == nil {
			t.Errorf("%d%% for %d minutes was built", tc.percent, tc.minutes)
		}
	}
	for _, tc := range []struct{ minutes, percent int32 }{{15, 0}, {4320, 250}} {
		if _, err := control.NewSetTempRateRequest(tc.minutes, tc.percent); err != nil {
			t.Errorf("%d%% for %d minutes: %v", tc.percent, tc.minutes, err)
		}
	}
}

func TestSetTempRateRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetTempRateRequest")
}
