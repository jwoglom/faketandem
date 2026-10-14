package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCurrentBatteryV2Request(t *testing.T) {
	parity.TestMessage(t, "0005900500504b", 5, 1, messages.CurrentStatus, currentstatus.NewCurrentBatteryV2Request())
}

func TestCurrentBatteryV2RequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CurrentBatteryV2Request")
}
