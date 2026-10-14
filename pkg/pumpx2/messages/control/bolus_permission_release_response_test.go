package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestBolusPermissionReleaseResponse(t *testing.T) {
	for _, tc := range []struct {
		name        string
		pairingCode string
		hex         string
		txID        int
		status      int32
		want        control.ReleaseStatus
	}{
		{"released", "6VeDeRAL5DCigGw2", "003af13a19003923851b8854859ea0fc17fe530b3c3c4208ae30a555356f0985", 58, 0, control.ReleaseStatusSuccess},
		{"released on another pump", "Ns7vSuFYcfTLAXmb", "0005f105190099abe11b341684fe037937a533833292ee3df0d4bc008b71fca4", 5, 0, control.ReleaseStatusSuccess},
		{"already released", "Ns7vSuFYcfTLAXmb", "0006f1061901a2abe11b6fbc54b70597cb6d423d18d162cd350efcfcb9f5a16a", 6, 1, control.ReleaseStatusFailure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tester := parity.MessageTester{Env: messages.Env{PairingCode: tc.pairingCode, TimeSinceReset: "461710079"}}
			parsed := tester.Test(t, tc.hex, tc.txID, 1, messages.Control,
				control.NewBolusPermissionReleaseResponse(tc.status)).(*control.BolusPermissionReleaseResponse)
			if s := parsed.ReleaseStatus(); s == nil || *s != tc.want {
				t.Errorf("release status %v, want %v", s, tc.want)
			}
		})
	}
}

func TestBolusPermissionReleaseResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "BolusPermissionReleaseResponse")
}
