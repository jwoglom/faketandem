package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestPlaySoundResponse(t *testing.T) {
	expected, err := control.NewPlaySoundResponseFromCargo([]byte{0})
	if err != nil {
		t.Fatal(err)
	}
	parity.TestMessage(t, "00cef5ce1900cef0f21fbf00d7c6462ba60db8d201b730849ce4458962818856", -50, 1, messages.Control, expected)
}

func TestPlaySoundResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "PlaySoundResponse")
}
