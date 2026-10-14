package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCurrentBatteryV1Request(t *testing.T) {
	parity.TestMessage(t, "0003340300aa80", 3, 1, messages.CurrentStatus, currentstatus.NewCurrentBatteryV1Request())
}

func TestCurrentBatteryV1RequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CurrentBatteryV1Request")
}
