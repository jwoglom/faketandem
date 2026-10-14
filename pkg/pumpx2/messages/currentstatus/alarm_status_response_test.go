package currentstatus_test

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestAlarmStatusResponseWithNoAlarms(t *testing.T) {
	parsed := parity.TestMessage(t, "000347030800000000000000005721", 3, 1, messages.CurrentStatus,
		currentstatus.NewAlarmStatusResponse(big.NewInt(0))).(*currentstatus.AlarmStatusResponse)
	if parsed.Size() != 0 {
		t.Errorf("alarms %v", parsed.Alarms)
	}
}

func TestAlarmStatusResponseAfterPumpReset(t *testing.T) {
	parsed := parity.TestMessage(t, "000c470c0808008000000000001cbd", 12, 1, messages.CurrentStatus,
		currentstatus.NewAlarmStatusResponseFromAlarms(currentstatus.AlarmResponseTypePumpResetAlarm, currentstatus.AlarmResponseTypeResumePumpAlarm2),
	).(*currentstatus.AlarmStatusResponse)
	want := []currentstatus.AlarmResponseType{currentstatus.AlarmResponseTypePumpResetAlarm, currentstatus.AlarmResponseTypeResumePumpAlarm2}
	if !reflect.DeepEqual(parsed.Alarms, want) {
		t.Errorf("alarms %v, want %v", parsed.Alarms, want)
	}
	if ids := parsed.NotificationIDs(); !reflect.DeepEqual(ids, []int32{3, 23}) {
		t.Errorf("notification IDs %v", ids)
	}
}

func TestAlarmStatusResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "AlarmStatusResponse")
}
