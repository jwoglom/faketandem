package control_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetConfigurableFeatureResponse(t *testing.T) {
	for _, tc := range []struct {
		name           string
		status, reason int32
	}{
		{"success", 0, 0},
		{"error", 1, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := control.NewSetConfigurableFeatureResponse(tc.status, tc.reason)
			parsed := &control.SetConfigurableFeatureResponse{}
			if err := parsed.Parse(expected.Cargo()); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(parsed.Cargo(), expected.Cargo()) || len(parsed.Cargo()) != 2 {
				t.Errorf("cargo %x, want %x", parsed.Cargo(), expected.Cargo())
			}
			if parsed.Status != tc.status || parsed.Reason != tc.reason {
				t.Errorf("status %d reason %d, want %d reason %d", parsed.Status, parsed.Reason, tc.status, tc.reason)
			}
		})
	}
}

func TestSetConfigurableFeatureResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetConfigurableFeatureResponse")
}
