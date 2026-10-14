package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCancelBolusRequest(t *testing.T) {
	for _, tc := range []struct {
		bolusID int32
		txID    int
		packets []string
	}{
		{10677, 0xbc, []string{"01bca0bc1cb52900004123851bde7e214f942297", "00bcea477474bb87dbb3293c92642f43e8"}},
		{10678, 0xe8, []string{"01e8a0e81cb62900007a23851b1b100405c517c0", "00e887856f69ef78c79bf186fa39a0c5d1"}},
	} {
		tester := parity.MessageTester{Env: messages.Env{PairingCode: "6VeDeRAL5DCigGw2", TimeSinceReset: "461710145"}}
		tester.Test(t, tc.packets[0], tc.txID, 1, messages.Control, control.NewCancelBolusRequest(tc.bolusID), tc.packets[1:]...)
	}
}

func TestCancelBolusRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CancelBolusRequest")
}
