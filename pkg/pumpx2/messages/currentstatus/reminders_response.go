package currentstatus

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/models"
)

// RemindersResponse is the reminders configured on the pump: low and high
// BG, site change, four missed meal bolus windows, after bolus BG check and
// additional bolus. The low and high BG thresholds are in mg/dL, and
// SiteChangeDays is how many days after the last site change the site change
// reminder goes off. Status is 0 on success.
type RemindersResponse struct {
	messages.Base
	LowBGReminder           Reminder `pumpx2:"lowBGReminder"`
	HighBGReminder          Reminder `pumpx2:"highBGReminder"`
	SiteChangeReminder      Reminder `pumpx2:"siteChangeReminder"`
	MissedBolusReminder0    Reminder `pumpx2:"missedBolusReminder0"`
	MissedBolusReminder1    Reminder `pumpx2:"missedBolusReminder1"`
	MissedBolusReminder2    Reminder `pumpx2:"missedBolusReminder2"`
	MissedBolusReminder3    Reminder `pumpx2:"missedBolusReminder3"`
	AfterBolusReminder      Reminder `pumpx2:"afterBolusReminder"`
	AdditionalBolusReminder Reminder `pumpx2:"additionalBolusReminder"`
	LowBGThreshold          int32    `pumpx2:"lowBGThreshold"`
	HighBGThreshold         int32    `pumpx2:"highBGThreshold"`
	SiteChangeDays          int32    `pumpx2:"siteChangeDays"`
	Status                  int32    `pumpx2:"status"`
}

var remindersResponseProps = messages.Props{OpCode: 89, Size: 105, Type: messages.Response, Request: "RemindersRequest"}

func init() {
	messages.Register("RemindersResponse", &RemindersResponse{}, remindersResponseProps,
		messages.Constructor{
			Params: "RemindersResponse.Reminder lowBGReminder, RemindersResponse.Reminder highBGReminder, RemindersResponse.Reminder siteChangeReminder, " +
				"RemindersResponse.Reminder missedBolusReminder0, RemindersResponse.Reminder missedBolusReminder1, " +
				"RemindersResponse.Reminder missedBolusReminder2, RemindersResponse.Reminder missedBolusReminder3, " +
				"RemindersResponse.Reminder afterBolusReminder, RemindersResponse.Reminder additionalBolusReminder, " +
				"int lowBGThreshold, int highBGThreshold, int siteChangeDays, int status",
			Func: NewRemindersResponse,
		},
	)
}

// NewRemindersResponse builds the response.
func NewRemindersResponse(lowBGReminder, highBGReminder, siteChangeReminder, missedBolusReminder0, missedBolusReminder1, missedBolusReminder2,
	missedBolusReminder3, afterBolusReminder, additionalBolusReminder Reminder, lowBGThreshold, highBGThreshold, siteChangeDays, status int32) *RemindersResponse {
	m := &RemindersResponse{
		LowBGReminder:           lowBGReminder,
		HighBGReminder:          highBGReminder,
		SiteChangeReminder:      siteChangeReminder,
		MissedBolusReminder0:    missedBolusReminder0,
		MissedBolusReminder1:    missedBolusReminder1,
		MissedBolusReminder2:    missedBolusReminder2,
		MissedBolusReminder3:    missedBolusReminder3,
		AfterBolusReminder:      afterBolusReminder,
		AdditionalBolusReminder: additionalBolusReminder,
		LowBGThreshold:          lowBGThreshold,
		HighBGThreshold:         highBGThreshold,
		SiteChangeDays:          siteChangeDays,
		Status:                  status,
	}
	m.SetCargo(helpers.Combine(
		lowBGReminder.Build(),
		highBGReminder.Build(),
		siteChangeReminder.Build(),
		missedBolusReminder0.Build(),
		missedBolusReminder1.Build(),
		missedBolusReminder2.Build(),
		missedBolusReminder3.Build(),
		afterBolusReminder.Build(),
		additionalBolusReminder.Build(),
		helpers.FirstTwoBytesLittleEndian(lowBGThreshold),
		helpers.FirstTwoBytesLittleEndian(highBGThreshold),
		[]byte{byte(siteChangeDays), byte(status)},
	))
	return m
}

// Parse implements messages.Message.
func (m *RemindersResponse) Parse(raw []byte) error {
	if len(raw) != remindersResponseProps.Size {
		return messages.InvalidLength("RemindersResponse", len(raw), remindersResponseProps.Size)
	}
	m.SetCargo(raw)
	m.LowBGReminder = ReadReminder(raw, 0)
	m.HighBGReminder = ReadReminder(raw, 11)
	m.SiteChangeReminder = ReadReminder(raw, 22)
	m.MissedBolusReminder0 = ReadReminder(raw, 33)
	m.MissedBolusReminder1 = ReadReminder(raw, 44)
	m.MissedBolusReminder2 = ReadReminder(raw, 55)
	m.MissedBolusReminder3 = ReadReminder(raw, 66)
	m.AfterBolusReminder = ReadReminder(raw, 77)
	m.AdditionalBolusReminder = ReadReminder(raw, 88)
	m.LowBGThreshold = helpers.ReadShort(raw, 99)
	m.HighBGThreshold = helpers.ReadShort(raw, 101)
	m.SiteChangeDays = int32(int8(raw[103]))
	m.Status = int32(int8(raw[104]))
	return nil
}

// Reminder is one of the pump's reminders, eleven bytes on the wire.
// Frequency is in minutes: how long after a BG or bolus the reminder goes
// off, or the site change reminder's time of day. StartTime and EndTime are
// minutes since midnight bounding a missed meal bolus reminder's window.
// pumpX2 does not know what ValidityStatus means.
type Reminder struct {
	Frequency int64 `pumpx2:"frequency"`
	StartTime int32 `pumpx2:"startTime"`
	EndTime   int32 `pumpx2:"endTime"`
	// ActiveDays is a models.MultiDay bitmask.
	ActiveDays     int32 `pumpx2:"activeDays"`
	Enabled        int32 `pumpx2:"enabled"`
	ValidityStatus int32 `pumpx2:"validityStatus"`
}

// ReadReminder reads the reminder at offset i of cargo.
func ReadReminder(cargo []byte, i int) Reminder {
	return Reminder{
		Frequency:      helpers.ReadUint32(cargo, i),
		StartTime:      helpers.ReadShort(cargo, i+4),
		EndTime:        helpers.ReadShort(cargo, i+6),
		ActiveDays:     int32(int8(cargo[i+8])),
		Enabled:        int32(int8(cargo[i+9])),
		ValidityStatus: int32(int8(cargo[i+10])),
	}
}

// Build is the reminder's eleven bytes.
func (r Reminder) Build() []byte {
	return helpers.Combine(
		helpers.ToUint32(r.Frequency),
		helpers.FirstTwoBytesLittleEndian(r.StartTime),
		helpers.FirstTwoBytesLittleEndian(r.EndTime),
		[]byte{byte(r.ActiveDays), byte(r.Enabled), byte(r.ValidityStatus)},
	)
}

// FrequencyTime is Frequency as hours and minutes, from its low 32 bits.
func (r Reminder) FrequencyTime() models.MinsTime { return models.NewMinsTime(int32(r.Frequency)) }

// Start is when the reminder's window starts.
func (r Reminder) Start() models.MinsTime { return models.NewMinsTime(r.StartTime) }

// End is when the reminder's window ends.
func (r Reminder) End() models.MinsTime { return models.NewMinsTime(r.EndTime) }

// Days are the days the reminder is active, as pumpX2 reads ActiveDays.
func (r Reminder) Days() []models.MultiDay { return models.MultiDayFromBitmask(r.ActiveDays) }

func (r Reminder) String() string {
	return messages.AutoToString("RemindersResponse.Reminder", r)
}
