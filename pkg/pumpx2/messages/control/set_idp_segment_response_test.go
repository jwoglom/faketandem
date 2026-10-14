package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetIDPSegmentResponse(t *testing.T) {
	expected, err := control.NewSetIDPSegmentResponseFromCargo([]byte{0, 2})
	if err != nil {
		t.Fatal(err)
	}
	parity.TestMessage(t, "002dab2d1a000260033e205bc7dccaad0b38c91c4f40c451c67c31f90c26c1ccb4", 45, 1, messages.Control, expected)
}

func TestSetIDPSegmentResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetIDPSegmentResponse")
}
