package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCGMAlertStatusRequest(t *testing.T) {
	parity.TestMessage(t, "00034a0300a343", 3, 1, messages.CurrentStatus, currentstatus.NewCGMAlertStatusRequest())
}

func TestCGMAlertStatusRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CGMAlertStatusRequest")
}
