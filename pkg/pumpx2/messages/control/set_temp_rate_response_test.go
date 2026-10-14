package control_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

// The answer to a 101% temp rate for 15 minutes, which the history log
// records as temp rate 15.
const setTempRateResponseCapture = "0048a5481c000f0000c3a3ee1fae161cb447c0235205f7a59796d2457a779c39c91b1a"

func TestSetTempRateResponse(t *testing.T) {
	expected, err := control.NewSetTempRateResponseFromCargo([]byte{0, 15, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	parity.TestMessage(t, setTempRateResponseCapture, 72, 1, messages.Control, expected)
}

func TestSetTempRateResponseFromAStatusAndID(t *testing.T) {
	expected := control.NewSetTempRateResponse(0, 15)
	if !bytes.Equal(expected.Cargo(), []byte{0, 15, 0, 0}) {
		t.Errorf("cargo %x", expected.Cargo())
	}
	parsed := parity.TestMessage(t, setTempRateResponseCapture, 72, 1, messages.Control, expected).(*control.SetTempRateResponse)
	if parsed.Status != 0 || parsed.TempRateID != 15 {
		t.Errorf("parsed %+v", parsed)
	}
}

func TestSetTempRateResponseFailed(t *testing.T) {
	m, err := control.NewSetTempRateResponseFromCargo([]byte{1, 0, 0, 1})
	if err != nil {
		t.Fatal(err)
	}
	if m.Status != 1 {
		t.Errorf("status %d", m.Status)
	}
}

func TestSetTempRateResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetTempRateResponse")
}
