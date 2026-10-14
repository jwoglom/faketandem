package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// SetMissedMealBolusReminderRequest configures the missed meal bolus
// reminder in slot ReminderIndex. StartTime and EndTime are minutes since
// midnight; SelectedDays has a bit per day, Monday first. Bitmask flags the
// fields the pump applies: 1 EnabledReminder, 2 StartTime, 4 EndTime,
// 8 SelectedDays.
type SetMissedMealBolusReminderRequest struct {
	messages.Base
	Bitmask         int32 `pumpx2:"bitmask"`
	EnabledReminder bool  `pumpx2:"enabledReminder"`
	EndTime         int32 `pumpx2:"endTime"`
	ReminderIndex   int32 `pumpx2:"reminderIndex"`
	SelectedDays    int32 `pumpx2:"selectedDays"`
	StartTime       int32 `pumpx2:"startTime"`
}

var setMissedMealBolusReminderRequestProps = messages.Props{OpCode: -38, Size: 8, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "SetMissedMealBolusReminderResponse"}

func init() {
	messages.Register("SetMissedMealBolusReminderRequest", &SetMissedMealBolusReminderRequest{}, setMissedMealBolusReminderRequestProps,
		messages.Constructor{
			Params: "int reminderIndex, boolean enabledReminder, int startTime, int endTime, int selectedDays, int bitmask",
			Func:   NewSetMissedMealBolusReminderRequest,
		},
	)
}

// NewSetMissedMealBolusReminderRequest builds the request.
func NewSetMissedMealBolusReminderRequest(reminderIndex int32, enabledReminder bool, startTime, endTime, selectedDays, bitmask int32) *SetMissedMealBolusReminderRequest {
	m := &SetMissedMealBolusReminderRequest{
		ReminderIndex:   reminderIndex,
		EnabledReminder: enabledReminder,
		StartTime:       startTime,
		EndTime:         endTime,
		SelectedDays:    selectedDays,
		Bitmask:         bitmask,
	}
	m.SetCargo(helpers.Combine(
		[]byte{byte(reminderIndex), helpers.Bool(enabledReminder)},
		helpers.FirstTwoBytesLittleEndian(startTime),
		helpers.FirstTwoBytesLittleEndian(endTime),
		[]byte{byte(selectedDays), byte(bitmask)},
	))
	return m
}

// Parse implements messages.Message.
func (m *SetMissedMealBolusReminderRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setMissedMealBolusReminderRequestProps, raw)
	if len(raw) != setMissedMealBolusReminderRequestProps.Size {
		return messages.InvalidLength("SetMissedMealBolusReminderRequest", len(raw), setMissedMealBolusReminderRequestProps.Size)
	}
	m.SetCargo(raw)
	m.ReminderIndex = int32(raw[0])
	m.EnabledReminder = raw[1] != 0
	m.StartTime = helpers.ReadShort(raw, 2)
	m.EndTime = helpers.ReadShort(raw, 4)
	m.SelectedDays = int32(raw[6])
	m.Bitmask = int32(raw[7])
	return nil
}
