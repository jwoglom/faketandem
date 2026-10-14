package control_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestJoinFSL3Response(t *testing.T) {
	for _, tc := range []struct {
		name               string
		status, statusType int32
	}{
		{"success", 0, 0},
		{"error", 1, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := control.NewJoinFSL3Response(tc.status, tc.statusType)
			parsed := &control.JoinFSL3Response{}
			if err := parsed.Parse(expected.Cargo()); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(parsed.Cargo(), expected.Cargo()) || len(parsed.Cargo()) != 2 {
				t.Errorf("cargo %x, want %x", parsed.Cargo(), expected.Cargo())
			}
			if parsed.Status != tc.status || parsed.StatusType != tc.statusType {
				t.Errorf("status %d type %d, want %d type %d", parsed.Status, parsed.StatusType, tc.status, tc.statusType)
			}
		})
	}
}

func TestJoinFSL3ResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "JoinFSL3Response")
}
