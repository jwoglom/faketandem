package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// The reminders a SetBgReminderRequest's ReminderType names.
const (
	BgReminderTypeHigh       = 0
	BgReminderTypeLow        = 1
	BgReminderTypeAfterBolus = 2
)

// SetBgReminderRequest sets the BG reminder ReminderType names (a
// BgReminderType). ReminderThreshold is in mg/dL and ReminderMinutes in
// minutes. Bitmask says which settings change: bit 0 EnabledBGReminder, bit 1
// ReminderThreshold, bit 2 ReminderMinutes.
type SetBgReminderRequest struct {
	messages.Base
	ReminderType      int32 `pumpx2:"reminderType"`
	EnabledBGReminder bool  `pumpx2:"enabledBGReminder"`
	ReminderThreshold int32 `pumpx2:"reminderThreshold"`
	ReminderMinutes   int64 `pumpx2:"reminderMinutes"`
	Bitmask           int32 `pumpx2:"bitmask"`
}

var setBgReminderRequestProps = messages.Props{OpCode: -40, Size: 9, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "SetBgReminderResponse"}

func init() {
	messages.Register("SetBgReminderRequest", &SetBgReminderRequest{}, setBgReminderRequestProps,
		messages.Constructor{
			Params: "int reminderType, boolean enabledBGReminder, int reminderThreshold, long reminderMinutes, int bitmask",
			Func:   NewSetBgReminderRequest,
		},
	)
}

// NewSetBgReminderRequest builds the request.
func NewSetBgReminderRequest(reminderType int32, enabledBGReminder bool, reminderThreshold int32, reminderMinutes int64, bitmask int32) *SetBgReminderRequest {
	m := &SetBgReminderRequest{
		ReminderType:      reminderType,
		EnabledBGReminder: enabledBGReminder,
		ReminderThreshold: reminderThreshold,
		ReminderMinutes:   reminderMinutes,
		Bitmask:           bitmask,
	}
	m.SetCargo(helpers.Combine(
		[]byte{byte(reminderType), helpers.Bool(enabledBGReminder)},
		helpers.FirstTwoBytesLittleEndian(reminderThreshold),
		helpers.ToUint32(reminderMinutes),
		[]byte{byte(bitmask)},
	))
	return m
}

// Parse implements messages.Message.
func (m *SetBgReminderRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setBgReminderRequestProps, raw)
	if len(raw) != setBgReminderRequestProps.Size {
		return messages.InvalidLength("SetBgReminderRequest", len(raw), setBgReminderRequestProps.Size)
	}
	m.SetCargo(raw)
	m.ReminderType = int32(raw[0])
	m.EnabledBGReminder = raw[1] != 0
	m.ReminderThreshold = helpers.ReadShort(raw, 2)
	m.ReminderMinutes = helpers.ReadUint32(raw, 4)
	m.Bitmask = int32(raw[8])
	return nil
}
