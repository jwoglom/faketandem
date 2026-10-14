package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCGMHardwareInfoRequest(t *testing.T) {
	parity.TestMessage(t, "0004600400339b", 4, 1, messages.CurrentStatus, currentstatus.NewCGMHardwareInfoRequest())
}

func TestCGMHardwareInfoRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CGMHardwareInfoRequest")
}
