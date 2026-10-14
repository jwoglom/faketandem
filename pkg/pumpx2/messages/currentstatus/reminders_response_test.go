package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/models"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func reminderHourMins(hour int32) int32 {
	return models.MinsTime{Hour: hour}.Encode()
}

func TestRemindersResponse(t *testing.T) {
	off := currentstatus.Reminder{}
	siteChange := currentstatus.Reminder{Frequency: 1260, ValidityStatus: 3}
	afterBolus := currentstatus.Reminder{Frequency: 90}
	additionalBolus := currentstatus.Reminder{Enabled: 1}
	lowBG := currentstatus.Reminder{Frequency: 20, Enabled: 1, ValidityStatus: 3}
	highBG := currentstatus.Reminder{Frequency: int64(reminderHourMins(3)), Enabled: 1, ValidityStatus: 3}
	mondayEvening := currentstatus.Reminder{
		StartTime: reminderHourMins(12 + 5), EndTime: reminderHourMins(12 + 7),
		ActiveDays: models.MultiDayToBitmask(models.Monday), Enabled: 1, ValidityStatus: 30,
	}
	weekdayAfternoon := currentstatus.Reminder{
		StartTime: reminderHourMins(12 + 2), EndTime: reminderHourMins(12 + 3),
		ActiveDays: models.MultiDayToBitmask(models.Monday, models.Tuesday, models.Wednesday, models.Thursday, models.Friday),
		Enabled:    1, ValidityStatus: 30,
	}
	weekdayAfternoonOff := weekdayAfternoon
	weekdayAfternoonOff.Enabled = 0

	for _, tc := range []struct {
		name                        string
		hex                         string
		txID                        int
		lowBG, highBG               currentstatus.Reminder
		missed0, missed1            currentstatus.Reminder
		lowThreshold, highThreshold int32
		status                      int32
	}{
		{"all off, site change Thursday 9pm",
			"00035903690f000000000000000000007800000000000000000000ec0400000000000000000300000000000000000000000000000000000000000000000000000000000000000000000000000000000000005a0000000000000000000000000000000000000001004600c8000304e7f9",
			3, currentstatus.Reminder{Frequency: 15}, currentstatus.Reminder{Frequency: 120}, off, off, 70, 200, 4},
		{"low BG at 80 mg/dL after 20 minutes",
			"000459046914000000000000000001037800000000000000000000ec0400000000000000000300000000000000000000000000000000000000000000000000000000000000000000000000000000000000005a0000000000000000000000000000000000000001005000c80003052efa",
			4, lowBG, currentstatus.Reminder{Frequency: 120}, off, off, 80, 200, 5},
		{"high BG at 300 mg/dL after 3 hours",
			"00065906691400000000000000000103b400000000000000000103ec0400000000000000000300000000000000000000000000000000000000000000000000000000000000000000000000000000000000005a00000000000000000000000000000000000000010050002c0103077f31",
			6, lowBG, highBG, off, off, 80, 300, 7},
		{"missed bolus Monday 5-7pm and weekdays 2-3pm",
			"00075907691400000000000000000103b400000000000000000103ec0400000000000000000300000000fc03740401011e00000000480384031f011e000000000000000000000000000000000000000000005a00000000000000000000000000000000000000010050002c010307c5e8",
			7, lowBG, highBG, mondayEvening, weekdayAfternoon, 80, 300, 7},
		{"missed bolus Monday 5-7pm, weekdays 2-3pm off",
			"00085908691400000000000000000103b400000000000000000103ec0400000000000000000300000000fc03740401011e00000000480384031f001e000000000000000000000000000000000000000000005a00000000000000000000000000000000000000010050002c010307e289",
			8, lowBG, highBG, mondayEvening, weekdayAfternoonOff, 80, 300, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := currentstatus.NewRemindersResponse(tc.lowBG, tc.highBG, siteChange, tc.missed0, tc.missed1, off, off, afterBolus, additionalBolus,
				tc.lowThreshold, tc.highThreshold, 3, tc.status)
			parity.TestMessage(t, tc.hex, tc.txID, 7, messages.CurrentStatus, expected)
		})
	}
}

func TestRemindersResponseReadsAReminderWindow(t *testing.T) {
	r := currentstatus.ReadReminder([]byte{0, 0, 0, 0, 0xfc, 0x03, 0x74, 0x04, 1, 1, 30}, 0)
	if r.Start().String() != "17:00" || r.End().String() != "19:00" {
		t.Errorf("window %v-%v, want 17:00-19:00", r.Start(), r.End())
	}
	if days := r.Days(); len(days) != 1 || days[0] != models.Monday {
		t.Errorf("days %v, want [MONDAY]", days)
	}
}

func TestRemindersResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "RemindersResponse")
}
