package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// PrimeTubingSuspendResponse is the pump's answer to a
// PrimeTubingSuspendRequest: StatusCode 0 (ACK) means success. The byte
// between StatusCode and Reserve is padding.
type PrimeTubingSuspendResponse struct {
	messages.Base
	StatusCode int32 `pumpx2:"statusCode"`
	Reserve    int32 `pumpx2:"reserve"`
}

var primeTubingSuspendResponseProps = messages.Props{OpCode: -17, Size: 3, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "PrimeTubingSuspendRequest"}

func init() {
	messages.Register("PrimeTubingSuspendResponse", &PrimeTubingSuspendResponse{}, primeTubingSuspendResponseProps,
		messages.Constructor{Params: "int statusCode, int reserve", Func: NewPrimeTubingSuspendResponse},
	)
}

// NewPrimeTubingSuspendResponse builds the response.
func NewPrimeTubingSuspendResponse(statusCode, reserve int32) *PrimeTubingSuspendResponse {
	m := &PrimeTubingSuspendResponse{StatusCode: statusCode, Reserve: reserve}
	m.SetCargo([]byte{byte(statusCode), 0, byte(reserve)})
	return m
}

// Parse implements messages.Message.
func (m *PrimeTubingSuspendResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(primeTubingSuspendResponseProps, raw)
	if len(raw) != primeTubingSuspendResponseProps.Size {
		return messages.InvalidLength("PrimeTubingSuspendResponse", len(raw), primeTubingSuspendResponseProps.Size)
	}
	m.SetCargo(raw)
	m.StatusCode = int32(int8(raw[0]))
	m.Reserve = int32(int8(raw[2]))
	return nil
}
