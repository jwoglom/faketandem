package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestAlertStatusRequest(t *testing.T) {
	parity.TestMessage(t, "0003440300a258", 3, 1, messages.CurrentStatus, currentstatus.NewAlertStatusRequest())
}

func TestAlertStatusRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "AlertStatusRequest")
}
