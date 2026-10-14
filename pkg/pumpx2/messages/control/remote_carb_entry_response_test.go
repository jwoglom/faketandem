package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestRemoteCarbEntryResponse(t *testing.T) {
	for _, tc := range []struct {
		name           string
		timeSinceReset string
		hex            string
		txID           int
	}{
		{"bolus 10677", "461710145", "00aaf3aa19006023851b4f8ac061f1a3bfc05496c100ab5a42a8e06637ad1fc3", -86},
		{"bolus 10678", "461710202", "00cff3cf19008423851bdcb7bd0bcb44808c07d71cf37fa8e41b8c1d236a1df6", -49},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tester := parity.MessageTester{Env: messages.Env{PairingCode: "6VeDeRAL5DCigGw2", TimeSinceReset: tc.timeSinceReset}}
			tester.Test(t, tc.hex, tc.txID, 1, messages.Control, control.NewRemoteCarbEntryResponse(0))
		})
	}
}

func TestRemoteCarbEntryResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "RemoteCarbEntryResponse")
}
