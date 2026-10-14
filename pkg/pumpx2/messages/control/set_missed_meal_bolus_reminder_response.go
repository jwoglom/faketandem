package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetMissedMealBolusReminderResponse is the pump's answer to a
// SetMissedMealBolusReminderRequest: Status 0 is success.
type SetMissedMealBolusReminderResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var setMissedMealBolusReminderResponseProps = messages.Props{OpCode: -37, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "SetMissedMealBolusReminderRequest"}

func init() {
	messages.Register("SetMissedMealBolusReminderResponse", &SetMissedMealBolusReminderResponse{}, setMissedMealBolusReminderResponseProps,
		messages.Constructor{Params: "int status", Func: NewSetMissedMealBolusReminderResponse},
	)
}

// NewSetMissedMealBolusReminderResponse builds the response.
func NewSetMissedMealBolusReminderResponse(status int32) *SetMissedMealBolusReminderResponse {
	m := &SetMissedMealBolusReminderResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *SetMissedMealBolusReminderResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setMissedMealBolusReminderResponseProps, raw)
	if len(raw) != setMissedMealBolusReminderResponseProps.Size {
		return messages.InvalidLength("SetMissedMealBolusReminderResponse", len(raw), setMissedMealBolusReminderResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
