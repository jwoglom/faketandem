package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCancelBolusResponse(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		pairingCode                 string
		timeSinceReset              string
		hex                         string
		txID                        int
		statusID, bolusID, reasonID int32
		wantCancelled               bool
	}{
		{"bolus 10677", "6VeDeRAL5DCigGw2", "461710145", "00bca1bc1d00b52900006923851bda3537a211cbd8ae95fec6d7a2a3d892d8f8af36bdc7", 0xbc,
			0, 10677, 0, true},
		{"bolus 10678", "6VeDeRAL5DCigGw2", "461710145", "00e8a1e81d00b62900009e23851b90d69beb26e8071804420d24cee664931b862a875929", 0xe8,
			0, 10678, 0, true},
		{"after the bolus was delivered", "bJVYJq9jdtjxPcYa", "463692320", "0006a1061d01c02902004e62a31b93f40e8ae917aa40df366dd04d0cdb19d1c0fdd3b4f9", 6,
			1, 10688, 2, false},
		{"unknown bolus ID", "bJVYJq9jdtjxPcYa", "463693031", "0004a1041d019f8602000f65a31ba24ec318b5b102d56ee0db9463eb4d645ed2b2560905", 4,
			1, 34463, 2, false},
		// With no pairing code the parse ignores the signature, whose key the capture lacks.
		{"Mobi extended bolus", "", "1905413", "002da12d1d00f50100007b428b1e59cc3479f5c851559f897f196f9511a5916f7499ff65", 45,
			0, 501, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tester := parity.MessageTester{Env: messages.Env{PairingCode: tc.pairingCode, TimeSinceReset: tc.timeSinceReset}}
			parsed := tester.Test(t, tc.hex, tc.txID, 1, messages.Control,
				control.NewCancelBolusResponse(tc.statusID, tc.bolusID, tc.reasonID)).(*control.CancelBolusResponse)
			if got := parsed.WasCancelled(); got != tc.wantCancelled {
				t.Errorf("WasCancelled() = %v, want %v", got, tc.wantCancelled)
			}
			wantStatus, wantReason := control.CancelStatusSuccess, control.CancelReasonNoError
			if !tc.wantCancelled {
				wantStatus, wantReason = control.CancelStatusFailed, control.CancelReasonInvalidOrAlreadyDelivered
			}
			if s := parsed.Status(); s == nil || *s != wantStatus {
				t.Errorf("status %v, want %v", s, wantStatus)
			}
			if r := parsed.Reason(); r == nil || *r != wantReason {
				t.Errorf("reason %v, want %v", r, wantReason)
			}
		})
	}
}

func TestCancelBolusResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CancelBolusResponse")
}
