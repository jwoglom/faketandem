package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestTempRateRequest(t *testing.T) {
	parity.TestMessage(t, "00032a0300c8d8", 3, 1, messages.CurrentStatus, currentstatus.NewTempRateRequest())
}

func TestTempRateRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "TempRateRequest")
}
