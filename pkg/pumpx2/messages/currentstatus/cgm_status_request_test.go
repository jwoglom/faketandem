package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCGMStatusRequest(t *testing.T) {
	parity.TestMessage(t, "000350030001c7", 3, 1, messages.CurrentStatus, currentstatus.NewCGMStatusRequest())
}

func TestCGMStatusRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CGMStatusRequest")
}
