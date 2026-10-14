package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestInsulinStatusRequest(t *testing.T) {
	parity.TestMessage(t, "00042404005e5a", 4, 1, messages.CurrentStatus, currentstatus.NewInsulinStatusRequest())
}

func TestInsulinStatusRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "InsulinStatusRequest")
}
