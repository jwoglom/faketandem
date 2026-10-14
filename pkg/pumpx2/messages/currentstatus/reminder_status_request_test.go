package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestReminderStatusRequest(t *testing.T) {
	parity.TestMessage(t, "0003480300c32d", 3, 1, messages.CurrentStatus, currentstatus.NewReminderStatusRequest())
}

func TestReminderStatusRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ReminderStatusRequest")
}
