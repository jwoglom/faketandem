package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestAlarmStatusRequest(t *testing.T) {
	parity.TestMessage(t, "0003460300c236", 3, 1, messages.CurrentStatus, currentstatus.NewAlarmStatusRequest())
}

func TestAlarmStatusRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "AlarmStatusRequest")
}
