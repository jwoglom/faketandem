package control_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetRuntimeConfigResponseRoundTrips(t *testing.T) {
	for _, status := range []int32{0, 1} {
		expected := control.NewSetRuntimeConfigResponse(status)
		parsed := &control.SetRuntimeConfigResponse{}
		if err := parsed.Parse(expected.Cargo()); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(parsed.Cargo(), expected.Cargo()) || len(parsed.Cargo()) != 1 {
			t.Errorf("cargo %x, want %x", parsed.Cargo(), expected.Cargo())
		}
		if parsed.Status != status {
			t.Errorf("status %d, want %d", parsed.Status, status)
		}
	}
}

func TestSetRuntimeConfigResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetRuntimeConfigResponse")
}
