package control_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestUserInteractionResponse(t *testing.T) {
	parsed := &control.UserInteractionResponse{}
	if err := parsed.Parse([]byte{0}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(parsed.Cargo(), []byte{0}) || parsed.Status != 0 {
		t.Errorf("parsed %+v", parsed)
	}
}

func TestUserInteractionResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "UserInteractionResponse")
}
