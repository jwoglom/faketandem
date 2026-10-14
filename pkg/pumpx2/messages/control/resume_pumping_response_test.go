package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestResumePumpingResponse(t *testing.T) {
	parity.TestMessage(t, "003d9b3d1900b5a3ee1f16c41eadee9847ab861cd24604a11e4a68d0a7dd2abe", 61, 1, messages.Control,
		control.NewResumePumpingResponse(0))
}

func TestResumePumpingResponseKeepsTheSignedLowByteOfItsStatus(t *testing.T) {
	if got := control.NewResumePumpingResponse(200).Status; got != -56 {
		t.Errorf("status %d, want -56", got)
	}
}

func TestResumePumpingResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ResumePumpingResponse")
}
