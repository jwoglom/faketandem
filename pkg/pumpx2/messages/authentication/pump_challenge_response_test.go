package authentication_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/authentication"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestPumpChallengeResponse(t *testing.T) {
	for _, tc := range []struct {
		name    string
		hex     string
		success bool
	}{
		{"success", "0001130103010001e8cc", true},
		{"failure", "0001130103010000c9dc", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed := parity.TestMessage(t, tc.hex, 1, 1, messages.Authorization,
				authentication.NewPumpChallengeResponse(1, tc.success)).(*authentication.PumpChallengeResponse)
			if parsed.AppInstanceID != 1 || parsed.Success != tc.success {
				t.Errorf("parsed %+v", parsed)
			}
		})
	}
}

func TestPumpChallengeResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "PumpChallengeResponse")
}
