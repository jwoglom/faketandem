package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestStreamDataPreflightResponseTypes(t *testing.T) {
	m := control.NewStreamDataPreflightResponse(1, 2, 0)
	if m.StatusType() != control.PreflightStatusTypeAnotherCgmActive || m.StreamType() != control.PreflightStreamTypeFreestyleLibre2 {
		t.Errorf("%v, %v", m.StatusType(), m.StreamType())
	}
	if m := control.NewStreamDataPreflightResponse(1, 99, 7); m.StatusType() != control.PreflightStatusTypeUnknown || m.StreamType() != control.PreflightStreamTypeUnknown {
		t.Errorf("unknown IDs: %v, %v", m.StatusType(), m.StreamType())
	}
}

func TestStreamDataPreflightResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "StreamDataPreflightResponse")
}
