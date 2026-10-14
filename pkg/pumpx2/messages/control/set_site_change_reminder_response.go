package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetSiteChangeReminderResponse is the pump's answer to a
// SetSiteChangeReminderRequest: Status 0 is success.
type SetSiteChangeReminderResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var setSiteChangeReminderResponseProps = messages.Props{OpCode: -35, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "SetSiteChangeReminderRequest"}

func init() {
	messages.Register("SetSiteChangeReminderResponse", &SetSiteChangeReminderResponse{}, setSiteChangeReminderResponseProps,
		messages.Constructor{Params: "int status", Func: NewSetSiteChangeReminderResponse},
	)
}

// NewSetSiteChangeReminderResponse builds the response.
func NewSetSiteChangeReminderResponse(status int32) *SetSiteChangeReminderResponse {
	m := &SetSiteChangeReminderResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *SetSiteChangeReminderResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setSiteChangeReminderResponseProps, raw)
	if len(raw) != setSiteChangeReminderResponseProps.Size {
		return messages.InvalidLength("SetSiteChangeReminderResponse", len(raw), setSiteChangeReminderResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
