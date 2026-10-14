package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/models"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestControlIQSleepScheduleResponse(t *testing.T) {
	everyNight := currentstatus.SleepSchedule{Enabled: 1, ActiveDays: 127, StartTime: 90, EndTime: 600}
	unset := currentstatus.SleepSchedule{StartTime: 1380, EndTime: 420}
	parity.TestMessage(t, "00036b0318017f5a00580200006405a40100006405a40100006405a401a330", 3, 2, messages.CurrentStatus,
		currentstatus.NewControlIQSleepScheduleResponse(everyNight, unset, unset, unset))
}

func TestControlIQSleepScheduleResponseWeekdaysAndSaturday(t *testing.T) {
	weekdays := currentstatus.SleepSchedule{
		Enabled:    1,
		ActiveDays: models.MultiDayToBitmask(models.Monday, models.Tuesday, models.Wednesday, models.Thursday, models.Friday),
		StartTime:  models.MinsTime{Hour: 22, Min: 30}.Encode(),
		EndTime:    models.MinsTime{Hour: 10}.Encode(),
	}
	saturday := currentstatus.SleepSchedule{
		Enabled:    1,
		ActiveDays: models.MultiDayToBitmask(models.Saturday),
		StartTime:  models.MinsTime{Hour: 23}.Encode(),
		EndTime:    models.MinsTime{Hour: 7}.Encode(),
	}
	unset := currentstatus.SleepSchedule{StartTime: models.MinsTime{Hour: 23}.Encode(), EndTime: models.MinsTime{Hour: 7}.Encode()}
	parsed := parity.TestMessage(t, "00046b0418011f4605580201206405a40100006405a40100006405a40109de", 4, 2, messages.CurrentStatus,
		currentstatus.NewControlIQSleepScheduleResponse(weekdays, saturday, unset, unset)).(*currentstatus.ControlIQSleepScheduleResponse)
	if got := parsed.Schedule0.Start().String(); got != "22:30" {
		t.Errorf("first schedule starts at %s", got)
	}
}

func TestControlIQSleepScheduleResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ControlIQSleepScheduleResponse")
}
