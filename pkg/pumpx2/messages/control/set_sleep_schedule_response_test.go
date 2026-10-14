package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetSleepScheduleResponse(t *testing.T) {
	expected, err := control.NewSetSleepScheduleResponseFromCargo([]byte{0})
	if err != nil {
		t.Fatal(err)
	}
	tester := parity.MessageTester{Env: messages.Env{TimeSinceReset: "1905836"}}
	parsed := tester.Test(t, "00a2cfa2190099438b1e67bcbb479d9ba61f4ce07e3bfebf2305b87092ed11b4", -94, 1, messages.Control, expected).(*control.SetSleepScheduleResponse)
	if parsed.Status != 0 {
		t.Errorf("status %d", parsed.Status)
	}
}

func TestSetSleepScheduleResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetSleepScheduleResponse")
}
