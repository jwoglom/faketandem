package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCreateIDPRequest(t *testing.T) {
	newProfile, err := control.NewCreateIDPRequest("testprofile", 3000, 1000, 100, 2, 300, 1)
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := control.NewCreateIDPRequestCopying("dup", 1)
	if err != nil {
		t.Fatal(err)
	}
	profCDup, err := control.NewCreateIDPRequestCopying("ProfCDup", 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name           string
		timeSinceReset string
		txID           int
		packets        []string
		expected       *control.CreateIDPRequest
	}{
		{"new profile", "1", 0xc9, []string{"03c9e6c93b7465737470726f66696c6500000000", "02c90000b80b00000000e803640002002c011f05",
			"01c9ff01b5483e20f4256e953cbe7320b25c5de0", "00c9f9e8fc891764dd51b865"}, newProfile},
		{"copy of profile 1", "0", 55, []string{"0337e6373b647570000000000000000000000000", "0237000000000000000000000000000000000000",
			"01370100b0493e20b795d5f162b92bbd30cd8823", "00372bc754ba29764a5b4765"}, duplicate},
		{"copy of profile 2", "1", 104, []string{"0368e6683b50726f664344757000000000000000", "0268000000000000000000000000000000000000",
			"01680200d648472076197caf3096ac0d1c300758", "00684ef8851d5c1d36b8bd09"}, profCDup},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tester := parity.MessageTester{Env: messages.Env{TimeSinceReset: tc.timeSinceReset}}
			tester.Test(t, tc.packets[0], tc.txID, 4, messages.Control, tc.expected, tc.packets[1:]...)
		})
	}
}

func TestCreateIDPRequestRefusesANameLongerThan17Bytes(t *testing.T) {
	if _, err := control.NewCreateIDPRequestCopying("profile name of 18", 1); err == nil {
		t.Error("an 18-byte name was built")
	}
}

func TestCreateIDPRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CreateIDPRequest")
}
