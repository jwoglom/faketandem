package currentstatus_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestGetSavedG7PairingCodeResponse(t *testing.T) {
	expected := currentstatus.NewGetSavedG7PairingCodeResponse(7681)
	parsed := &currentstatus.GetSavedG7PairingCodeResponse{}
	if err := parsed.Parse([]byte{0x01, 0x1e}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(parsed.Cargo(), expected.Cargo()) {
		t.Errorf("cargo %x, want %x", parsed.Cargo(), expected.Cargo())
	}
	if parsed.PairingCode != 7681 {
		t.Errorf("pairing code %d", parsed.PairingCode)
	}
}

func TestGetSavedG7PairingCodeResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "GetSavedG7PairingCodeResponse")
}
