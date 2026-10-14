package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestDeleteIDPResponse(t *testing.T) {
	for _, tc := range []struct {
		name string
		hex  string
		txID int
	}{
		{"profile 2", "004daf4d1a000291033e208345bb76cb9e5327ede6e87a0e590223f1ab75c1f93d", 77},
		// pumpX2's capture of deleting profile 1 also reports ID 2.
		{"profile 1", "0053af531a0002a0033e20d85eaa5d3690b58e62edeac041a420ef93de1bdda8bf", 83},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parity.TestMessage(t, tc.hex, tc.txID, 1, messages.Control, control.NewDeleteIDPResponse(0, 2))
		})
	}
}

func TestDeleteIDPResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "DeleteIDPResponse")
}
