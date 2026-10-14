package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestPumpVersionResponse(t *testing.T) {
	for _, tc := range []struct {
		name string
		hex  string
		txID int
		want func() (*currentstatus.PumpVersionResponse, error)
	}{
		{"t:slim X2", "0003550330ac9d0100ac9d01000000000000000000e3c86505df560f0030000000000000005044db404100000000000000a2430f008722", 3,
			func() (*currentstatus.PumpVersionResponse, error) {
				return currentstatus.NewPumpVersionResponse(105900, 105900, 0, 0, 90556643, 1005279, "0", 1088111696, "A", 1000354)
			}},
		{"Mobi", "001e551e309d8849d8000000000000000000000000e0b8120035750f003000000000000000adb8de0d3000000000000000e0510f00d0e6", 30,
			func() (*currentstatus.PumpVersionResponse, error) {
				return currentstatus.NewPumpVersionResponse(3628697757, 0, 0, 0, 1226976, 1013045, "0", 232700077, "0", 1004000)
			}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := tc.want()
			if err != nil {
				t.Fatal(err)
			}
			parity.TestMessage(t, tc.hex, tc.txID, 3, messages.CurrentStatus, expected)
		})
	}
}

func TestPumpVersionResponseRefusesARevisionLongerThanEightBytes(t *testing.T) {
	if _, err := currentstatus.NewPumpVersionResponse(0, 0, 0, 0, 0, 0, "Profile 1", 0, "", 0); err == nil {
		t.Error("a 9-byte pumpRev was built")
	}
}

func TestPumpVersionResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "PumpVersionResponse")
}
