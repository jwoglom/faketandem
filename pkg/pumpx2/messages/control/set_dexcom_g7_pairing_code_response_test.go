package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetDexcomG7PairingCodeResponse(t *testing.T) {
	parity.TestMessage(t, "00f9fdf91a00009346fd1f7a5eccab7fe004d6d23a4472643416ebd8b9fe8ff2bf", -7, 1, messages.Control,
		control.NewSetDexcomG7PairingCodeResponse(0))
}

func TestSetDexcomG7PairingCodeResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetDexcomG7PairingCodeResponse")
}
