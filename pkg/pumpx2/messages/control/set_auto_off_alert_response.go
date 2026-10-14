package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetAutoOffAlertResponse is the pump's answer to a SetAutoOffAlertRequest:
// Status 0 means success.
type SetAutoOffAlertResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var setAutoOffAlertResponseProps = messages.Props{OpCode: -31, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "SetAutoOffAlertRequest"}

func init() {
	messages.Register("SetAutoOffAlertResponse", &SetAutoOffAlertResponse{}, setAutoOffAlertResponseProps,
		messages.Constructor{Params: "int status", Func: NewSetAutoOffAlertResponse},
	)
}

// NewSetAutoOffAlertResponse builds the response.
func NewSetAutoOffAlertResponse(status int32) *SetAutoOffAlertResponse {
	m := &SetAutoOffAlertResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *SetAutoOffAlertResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setAutoOffAlertResponseProps, raw)
	if len(raw) != setAutoOffAlertResponseProps.Size {
		return messages.InvalidLength("SetAutoOffAlertResponse", len(raw), setAutoOffAlertResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
