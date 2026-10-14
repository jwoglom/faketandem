package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetPumpAlertSnoozeRequest(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled bool
		minutes int32
		packets []string
		txID    int
	}{
		{"on for 10 minutes", true, 10, []string{"0116d4161a010a0db76820eabe11bbaf518d5a06", "0016ef31958dafac47f38327cad7f7"}, 22},
		{"on for 20 minutes", true, 20, []string{"0118d4181a01142fb768207351c4815476eb350a", "00184b210fb8263ff36ec4e138732f"}, 24},
		{"off", false, 0, []string{"0131d4311a0000fbb8682038d7a22100f7646faa", "00316f9d827605f665814a30dbe72b"}, 49},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := control.NewSetPumpAlertSnoozeRequest(tc.enabled, tc.minutes)
			parity.TestMessage(t, tc.packets[0], tc.txID, 1, messages.Control, expected, tc.packets[1:]...)
		})
	}
}

func TestSetPumpAlertSnoozeRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetPumpAlertSnoozeRequest")
}
