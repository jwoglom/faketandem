package control

import (
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/helpers"
)

// SetPumpAlertSnoozeRequest turns the Mobi's alert snooze (a triple press of
// its button) on or off. SnoozeDurationMins is 10 or 20 when on, 0 when off.
type SetPumpAlertSnoozeRequest struct {
	messages.Base
	SnoozeDurationMins int32 `pumpx2:"snoozeDurationMins"`
	SnoozeEnabled      bool  `pumpx2:"snoozeEnabled"`
}

var setPumpAlertSnoozeRequestProps = messages.Props{OpCode: -44, Size: 2, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "SetPumpAlertSnoozeResponse"}

func init() {
	messages.Register("SetPumpAlertSnoozeRequest", &SetPumpAlertSnoozeRequest{}, setPumpAlertSnoozeRequestProps,
		messages.Constructor{Params: "boolean snoozeEnabled, int snoozeDurationMins", Func: NewSetPumpAlertSnoozeRequest},
	)
}

// NewSetPumpAlertSnoozeRequest builds the request.
func NewSetPumpAlertSnoozeRequest(snoozeEnabled bool, snoozeDurationMins int32) *SetPumpAlertSnoozeRequest {
	m := &SetPumpAlertSnoozeRequest{SnoozeEnabled: snoozeEnabled, SnoozeDurationMins: snoozeDurationMins}
	m.SetCargo([]byte{helpers.Bool(snoozeEnabled), byte(snoozeDurationMins)})
	return m
}

// Parse implements messages.Message.
func (m *SetPumpAlertSnoozeRequest) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setPumpAlertSnoozeRequestProps, raw)
	if len(raw) != setPumpAlertSnoozeRequestProps.Size {
		return messages.InvalidLength("SetPumpAlertSnoozeRequest", len(raw), setPumpAlertSnoozeRequestProps.Size)
	}
	m.SetCargo(raw)
	m.SnoozeEnabled = raw[0] == 1
	m.SnoozeDurationMins = int32(int8(raw[1]))
	return nil
}
