package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestInsulinStatusResponse(t *testing.T) {
	parity.TestMessage(t, "000425040400000023397d", 4, 1, messages.CurrentStatus, currentstatus.NewInsulinStatusResponse(0, 0, 35))
}

func TestInsulinStatusResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "InsulinStatusResponse")
}
