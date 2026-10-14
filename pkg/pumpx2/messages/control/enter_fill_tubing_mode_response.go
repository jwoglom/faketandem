package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// EnterFillTubingModeResponse is the pump's answer to an
// EnterFillTubingModeRequest: Status 0 means it entered fill tubing mode.
type EnterFillTubingModeResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var enterFillTubingModeResponseProps = messages.Props{OpCode: -107, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, Request: "EnterFillTubingModeRequest"}

func init() {
	messages.Register("EnterFillTubingModeResponse", &EnterFillTubingModeResponse{}, enterFillTubingModeResponseProps,
		messages.Constructor{Params: "int status", Func: NewEnterFillTubingModeResponse},
	)
}

// NewEnterFillTubingModeResponse builds the response.
func NewEnterFillTubingModeResponse(status int32) *EnterFillTubingModeResponse {
	m := &EnterFillTubingModeResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *EnterFillTubingModeResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(enterFillTubingModeResponseProps, raw)
	if len(raw) != enterFillTubingModeResponseProps.Size {
		return messages.InvalidLength("EnterFillTubingModeResponse", len(raw), enterFillTubingModeResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
