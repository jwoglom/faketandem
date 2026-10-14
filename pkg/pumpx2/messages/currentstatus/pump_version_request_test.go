package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestPumpVersionRequest(t *testing.T) {
	parity.TestMessage(t, "0003540300c11b", 3, 1, messages.CurrentStatus, currentstatus.NewPumpVersionRequest())
}

func TestPumpVersionRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "PumpVersionRequest")
}
