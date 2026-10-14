package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestBolusPermissionResponse(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		timeSinceReset        string
		hex                   string
		txID                  int
		status, bolusID, nack int32
		wantReason            control.NackReason
		wantPermissionGranted bool
	}{
		{"granted", "461510588", "00efa3ef1e009a29000000c317821b8a7f9a9bd41e220a20856446e2ff9c69df60ab62987c", 0xef,
			0, 10650, 0, control.NackReasonPermissionGranted, true},
		{"open on the pump", "461634498", "0004a3041e010000000003dcfb831b6decc93831b2a85debe64b35493ac44a60652af9e0a7", 4,
			1, 0, 3, control.NackReasonPumpHasPermission, false},
		{"deliveries stopped", "461634831", "0004a3041e01000000000115fd831ba02d482bc5e45a7aab99f79ce1fd55f57b6828467fce", 4,
			1, 0, 1, control.NackReasonInvalidPumpingState, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tester := parity.MessageTester{Env: messages.Env{PairingCode: "6VeDeRAL5DCigGw2", TimeSinceReset: tc.timeSinceReset}}
			parsed := tester.Test(t, tc.hex, tc.txID, 1, messages.Control,
				control.NewBolusPermissionResponse(tc.status, tc.bolusID, tc.nack)).(*control.BolusPermissionResponse)
			if r := parsed.NackReason(); r == nil || *r != tc.wantReason {
				t.Errorf("nack reason %v, want %v", r, tc.wantReason)
			}
			if got := parsed.IsPermissionGranted(); got != tc.wantPermissionGranted {
				t.Errorf("permission granted %v, want %v", got, tc.wantPermissionGranted)
			}
		})
	}
}

func TestBolusPermissionResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "BolusPermissionResponse")
}
