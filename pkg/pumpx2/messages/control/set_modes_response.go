package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetModesResponse is the pump's answer to a SetModesRequest: Status 0 is
// success.
type SetModesResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var setModesResponseProps = messages.Props{OpCode: -51, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, MinAPI: "MOBI_API_V3_5", SupportedDevices: "MOBI_ONLY", Request: "SetModesRequest"}

func init() {
	messages.Register("SetModesResponse", &SetModesResponse{}, setModesResponseProps,
		messages.Constructor{Params: "byte[] raw", Func: NewSetModesResponseFromCargo},
		messages.Constructor{Params: "int status", Func: NewSetModesResponse},
	)
}

// NewSetModesResponseFromCargo parses a response's payload.
func NewSetModesResponseFromCargo(raw []byte) (*SetModesResponse, error) {
	m := &SetModesResponse{}
	return m, m.Parse(raw)
}

// NewSetModesResponse builds the response. As in pumpX2, only the payload
// carries the status: the Status field stays 0.
func NewSetModesResponse(status int32) *SetModesResponse {
	m := &SetModesResponse{}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *SetModesResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setModesResponseProps, raw)
	if len(raw) != setModesResponseProps.Size {
		return messages.InvalidLength("SetModesResponse", len(raw), setModesResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
