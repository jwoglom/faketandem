package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestResumePumpingRequest(t *testing.T) {
	parity.TestMessage(t, "013d9a3d1808eaee1ff9e90b186d391c9892ca8f", 61, 1, messages.Control, control.NewResumePumpingRequest(), "003d7bc20e2a68cc7bc5c6fcce")
}

func TestResumePumpingRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ResumePumpingRequest")
}
