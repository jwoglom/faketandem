package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCreateIDPResponse(t *testing.T) {
	parity.TestMessage(t, "00c9e7c91a000286023e203400da47812a7ab10adc5f006d21fda44a7d7afa907b", 0xc9, 1, messages.Control,
		control.NewCreateIDPResponse(0, 2))
}

func TestCreateIDPResponseToACopy(t *testing.T) {
	expected, err := control.NewCreateIDPResponseFromCargo([]byte{0, 2})
	if err != nil {
		t.Fatal(err)
	}
	parity.TestMessage(t, "0037e7371a000281033e20e6b4f3960cd57ac9d1d4d84e18cd1571588aa0cf4281", 55, 1, messages.Control, expected)
}

func TestCreateIDPResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CreateIDPResponse")
}
