package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// FillCannulaResponse is the pump's answer to a FillCannulaRequest: Status 0
// means it took the request.
type FillCannulaResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var fillCannulaResponseProps = messages.Props{OpCode: -103, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, MinAPI: "MOBI_API_V3_5", SupportedDevices: "MOBI_ONLY", Request: "FillCannulaRequest"}

func init() {
	messages.Register("FillCannulaResponse", &FillCannulaResponse{}, fillCannulaResponseProps,
		messages.Constructor{Params: "int status", Func: NewFillCannulaResponse},
	)
}

// NewFillCannulaResponse builds the response.
func NewFillCannulaResponse(status int32) *FillCannulaResponse {
	m := &FillCannulaResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *FillCannulaResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(fillCannulaResponseProps, raw)
	if len(raw) != fillCannulaResponseProps.Size {
		return messages.InvalidLength("FillCannulaResponse", len(raw), fillCannulaResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
