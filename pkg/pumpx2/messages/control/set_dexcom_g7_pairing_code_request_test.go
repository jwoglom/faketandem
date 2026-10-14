package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetDexcomG7PairingCodeRequest(t *testing.T) {
	parsed := parity.TestMessage(t, "02f9fcf920da0d000000000000e68cfd1fc0e13b", -7, 1, messages.Control, control.NewSetDexcomG7PairingCodeRequest(3546),
		"01f9ff4ae4968d2f256e803caa2487aeea5e8e20", "00f9f8").(*control.SetDexcomG7PairingCodeRequest)
	if parsed.PairingCode != 3546 {
		t.Errorf("pairing code %d, want 3546", parsed.PairingCode)
	}
}

func TestSetDexcomG7PairingCodeRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetDexcomG7PairingCodeRequest")
}
