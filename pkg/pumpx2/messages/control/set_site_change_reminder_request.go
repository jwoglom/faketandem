package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// SetSiteChangeReminderRequest configures the site change reminder: every
// DayCount days at TimeOfDayMinutes, minutes since midnight. Bitmask flags
// the fields the pump applies: 1 Enable, 2 DayCount, 4 TimeOfDayMinutes.
// DayCount goes out as a single byte.
type SetSiteChangeReminderRequest struct {
	messages.Base
	Bitmask          int32 `pumpx2:"bitmask"`
	DayCount         int32 `pumpx2:"dayCount"`
	Enable           bool  `pumpx2:"enable"`
	TimeOfDayMinutes int64 `pumpx2:"timeOfDayMinutes"`
}

var setSiteChangeReminderRequestProps = messages.Props{OpCode: -36, Size: 7, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "SetSiteChangeReminderResponse"}

func init() {
	messages.Register("SetSiteChangeReminderRequest", &SetSiteChangeReminderRequest{}, setSiteChangeReminderRequestProps,
		messages.Constructor{Params: "boolean enable, int dayCount, long timeOfDayMinutes, int bitmask", Func: NewSetSiteChangeReminderRequest},
	)
}

// NewSetSiteChangeReminderRequest builds the request.
func NewSetSiteChangeReminderRequest(enable bool, dayCount int32, timeOfDayMinutes int64, bitmask int32) *SetSiteChangeReminderRequest {
	m := &SetSiteChangeReminderRequest{Enable: enable, DayCount: dayCount, TimeOfDayMinutes: timeOfDayMinutes, Bitmask: bitmask}
	m.SetCargo(helpers.Combine(
		[]byte{helpers.Bool(enable), byte(dayCount)},
		helpers.ToUint32(timeOfDayMinutes),
		[]byte{byte(bitmask)},
	))
	return m
}

// Parse implements messages.Message.
func (m *SetSiteChangeReminderRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setSiteChangeReminderRequestProps, raw)
	if len(raw) != setSiteChangeReminderRequestProps.Size {
		return messages.InvalidLength("SetSiteChangeReminderRequest", len(raw), setSiteChangeReminderRequestProps.Size)
	}
	m.SetCargo(raw)
	m.Enable = raw[0] != 0
	m.DayCount = int32(raw[1])
	m.TimeOfDayMinutes = helpers.ReadUint32(raw, 2)
	m.Bitmask = int32(raw[6])
	return nil
}
