package currentstatus_test

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestReminderStatusResponse(t *testing.T) {
	for _, tc := range []struct {
		name      string
		hex       string
		txID      int
		reminders []currentstatus.ReminderType
	}{
		{"none", "000349030800000000000000002d72", 3, []currentstatus.ReminderType{}},
		{"site change", "00194919080400000000000000177d", 25, []currentstatus.ReminderType{currentstatus.ReminderSiteChange}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := currentstatus.NewReminderStatusResponse(currentstatus.ReminderTypeToBitmask(tc.reminders...))
			if err != nil {
				t.Fatal(err)
			}
			parsed := parity.TestMessage(t, tc.hex, tc.txID, 1, messages.CurrentStatus, expected).(*currentstatus.ReminderStatusResponse)
			if got := parsed.Reminders(); !reflect.DeepEqual(got, tc.reminders) {
				t.Errorf("reminders %v, want %v", got, tc.reminders)
			}
		})
	}
}

func TestReminderStatusResponseNamesOnlyTheKnownReminders(t *testing.T) {
	bitmask := new(big.Int).SetUint64(1<<63 | 1<<9 | 1<<8)
	m, err := currentstatus.NewReminderStatusResponse(bitmask)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.NotificationIDs(); !reflect.DeepEqual(got, []int32{8}) {
		t.Errorf("notification IDs %v, want [8]", got)
	}
	if got := m.Reminders()[1].String(); got != "DEFAULT_REMINDER_9" {
		t.Errorf("bit 9 is %s", got)
	}
	if m.BitMap() >= 0 {
		t.Errorf("bit map %d, want bit 63 set", m.BitMap())
	}
}

func TestReminderStatusResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ReminderStatusResponse")
}
