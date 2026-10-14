package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// DismissNotificationResponse is the pump's answer to a
// DismissNotificationRequest: Status 0 is success.
type DismissNotificationResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var dismissNotificationResponseProps = messages.Props{OpCode: -71, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control, Request: "DismissNotificationRequest"}

func init() {
	messages.Register("DismissNotificationResponse", &DismissNotificationResponse{}, dismissNotificationResponseProps,
		messages.Constructor{Params: "int status", Func: NewDismissNotificationResponse},
	)
}

// NewDismissNotificationResponse builds the response.
func NewDismissNotificationResponse(status int32) *DismissNotificationResponse {
	m := &DismissNotificationResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *DismissNotificationResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(dismissNotificationResponseProps, raw)
	if len(raw) != dismissNotificationResponseProps.Size {
		return messages.InvalidLength("DismissNotificationResponse", len(raw), dismissNotificationResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
