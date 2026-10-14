package control_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetRuntimeConfigRequestRoundTrips(t *testing.T) {
	for _, tc := range []struct {
		name             string
		config0, config1 int32
		cargo            []byte
	}{
		{"what the app sends", 0, 0, []byte{0, 0}},
		{"other values", 1, 2, []byte{1, 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed := &control.SetRuntimeConfigRequest{}
			if err := parsed.Parse(control.NewSetRuntimeConfigRequest(tc.config0, tc.config1).Cargo()); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(parsed.Cargo(), tc.cargo) || parsed.Config0 != tc.config0 || parsed.Config1 != tc.config1 {
				t.Errorf("parsed %+v", parsed)
			}
		})
	}
}

func TestSetRuntimeConfigRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetRuntimeConfigRequest")
}
