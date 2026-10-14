package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestControlIQSleepScheduleRequest(t *testing.T) {
	parity.TestMessage(t, "00036a030065c5", 3, 1, messages.CurrentStatus, currentstatus.NewControlIQSleepScheduleRequest())
}

func TestControlIQSleepScheduleRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ControlIQSleepScheduleRequest")
}
