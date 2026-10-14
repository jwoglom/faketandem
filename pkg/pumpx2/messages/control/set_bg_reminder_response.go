package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetBgReminderResponse is the pump's answer to a SetBgReminderRequest:
// Status 0 means success.
type SetBgReminderResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var setBgReminderResponseProps = messages.Props{OpCode: -39, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "SetBgReminderRequest"}

func init() {
	messages.Register("SetBgReminderResponse", &SetBgReminderResponse{}, setBgReminderResponseProps,
		messages.Constructor{Params: "int status", Func: NewSetBgReminderResponse},
	)
}

// NewSetBgReminderResponse builds the response.
func NewSetBgReminderResponse(status int32) *SetBgReminderResponse {
	m := &SetBgReminderResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *SetBgReminderResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setBgReminderResponseProps, raw)
	if len(raw) != setBgReminderResponseProps.Size {
		return messages.InvalidLength("SetBgReminderResponse", len(raw), setBgReminderResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
