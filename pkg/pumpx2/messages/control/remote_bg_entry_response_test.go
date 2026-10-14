package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestRemoteBgEntryResponse(t *testing.T) {
	for _, tc := range []struct {
		name           string
		timeSinceReset string
		hex            string
		txID           int
	}{
		{"bolus 10676", "461710145", "003bb73b19003923851b0b8a07e99fa174bf0e01c333ae169f8d1e836f9f2792", 59},
		{"bolus 10677", "461710145", "00a9b7a919005f23851b79da6d43236154c7564648e5a88ab528951007dcf2c3", -87},
		{"bolus 10678", "461710202", "00ceb7ce19008323851b5a103ba6a9e9dc9ee8799d46bc88ffb6fefe9aaffe5f", -50},
		{"bolus 10652", "461589158", "003cb73c1900e54a831ba9ee8424e341a16f6efc75204b5bcb3c91256accd804", 60},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tester := parity.MessageTester{Env: messages.Env{PairingCode: "6VeDeRAL5DCigGw2", TimeSinceReset: tc.timeSinceReset}}
			parsed := tester.Test(t, tc.hex, tc.txID, 1, messages.Control, control.NewRemoteBgEntryResponse(0)).(*control.RemoteBgEntryResponse)
			if !parsed.IsSuccess() {
				t.Error("the BG was not taken")
			}
		})
	}
}

func TestRemoteBgEntryResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "RemoteBgEntryResponse")
}
