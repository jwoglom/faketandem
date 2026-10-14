package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/models"
)

// ControlIQSleepScheduleResponse is Control-IQ's four sleep schedules. The
// pump's screen shows the first two.
type ControlIQSleepScheduleResponse struct {
	messages.Base
	Schedule0 SleepSchedule `pumpx2:"schedule0"`
	Schedule1 SleepSchedule `pumpx2:"schedule1"`
	Schedule2 SleepSchedule `pumpx2:"schedule2"`
	Schedule3 SleepSchedule `pumpx2:"schedule3"`
}

var controlIQSleepScheduleResponseProps = messages.Props{OpCode: 107, Size: 24, Type: messages.Response, Request: "ControlIQSleepScheduleRequest"}

func init() {
	messages.Register("ControlIQSleepScheduleResponse", &ControlIQSleepScheduleResponse{}, controlIQSleepScheduleResponseProps,
		messages.Constructor{
			Params: "ControlIQSleepScheduleResponse.SleepSchedule schedule0, ControlIQSleepScheduleResponse.SleepSchedule schedule1, " +
				"ControlIQSleepScheduleResponse.SleepSchedule schedule2, ControlIQSleepScheduleResponse.SleepSchedule schedule3",
			Func: NewControlIQSleepScheduleResponse,
		},
	)
}

// NewControlIQSleepScheduleResponse builds the response from its schedules.
func NewControlIQSleepScheduleResponse(schedule0, schedule1, schedule2, schedule3 SleepSchedule) *ControlIQSleepScheduleResponse {
	m := &ControlIQSleepScheduleResponse{Schedule0: schedule0, Schedule1: schedule1, Schedule2: schedule2, Schedule3: schedule3}
	m.SetCargo(helpers.Combine(schedule0.Build(), schedule1.Build(), schedule2.Build(), schedule3.Build()))
	return m
}

// Parse implements messages.Message.
func (m *ControlIQSleepScheduleResponse) Parse(raw []byte) error {
	if len(raw) != controlIQSleepScheduleResponseProps.Size {
		return messages.InvalidLength("ControlIQSleepScheduleResponse", len(raw), controlIQSleepScheduleResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Schedule0 = ReadSleepSchedule(raw, 0)
	m.Schedule1 = ReadSleepSchedule(raw, 6)
	m.Schedule2 = ReadSleepSchedule(raw, 12)
	m.Schedule3 = ReadSleepSchedule(raw, 18)
	return nil
}

// SleepSchedule is one Control-IQ sleep schedule, six bytes on the wire.
type SleepSchedule struct {
	Enabled int32 `pumpx2:"enabled"`
	// ActiveDays is a models.MultiDay bitmask.
	ActiveDays int32 `pumpx2:"activeDays"`
	// StartTime and EndTime are minutes since midnight.
	StartTime int32 `pumpx2:"startTime"`
	EndTime   int32 `pumpx2:"endTime"`
}

// ReadSleepSchedule reads the schedule at offset i of cargo.
func ReadSleepSchedule(cargo []byte, i int) SleepSchedule {
	return SleepSchedule{
		Enabled:    int32(int8(cargo[i])),
		ActiveDays: int32(int8(cargo[i+1])),
		StartTime:  helpers.ReadShort(cargo, i+2),
		EndTime:    helpers.ReadShort(cargo, i+4),
	}
}

// Build is the schedule's six bytes.
func (s SleepSchedule) Build() []byte {
	return helpers.Combine(
		[]byte{byte(s.Enabled), byte(s.ActiveDays)},
		helpers.FirstTwoBytesLittleEndian(s.StartTime),
		helpers.FirstTwoBytesLittleEndian(s.EndTime),
	)
}

// Days are the days the schedule runs, as pumpX2 reads ActiveDays.
func (s SleepSchedule) Days() []models.MultiDay { return models.MultiDayFromBitmask(s.ActiveDays) }

// Start is when the schedule starts.
func (s SleepSchedule) Start() models.MinsTime { return models.NewMinsTime(s.StartTime) }

// End is when the schedule ends.
func (s SleepSchedule) End() models.MinsTime { return models.NewMinsTime(s.EndTime) }

func (s SleepSchedule) String() string {
	return messages.AutoToString("ControlIQSleepScheduleResponse.SleepSchedule", s)
}
