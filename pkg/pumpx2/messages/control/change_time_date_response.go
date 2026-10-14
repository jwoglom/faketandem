package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ChangeTimeDateResponse is the pump's answer to a ChangeTimeDateRequest:
// Status 0 is success.
type ChangeTimeDateResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var changeTimeDateResponseProps = messages.Props{OpCode: -41, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control, Request: "ChangeTimeDateRequest"}

func init() {
	messages.Register("ChangeTimeDateResponse", &ChangeTimeDateResponse{}, changeTimeDateResponseProps,
		messages.Constructor{Params: "int status", Func: NewChangeTimeDateResponse},
	)
}

// NewChangeTimeDateResponse builds the response.
func NewChangeTimeDateResponse(status int32) *ChangeTimeDateResponse {
	m := &ChangeTimeDateResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *ChangeTimeDateResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(changeTimeDateResponseProps, raw)
	if len(raw) != changeTimeDateResponseProps.Size {
		return messages.InvalidLength("ChangeTimeDateResponse", len(raw), changeTimeDateResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
