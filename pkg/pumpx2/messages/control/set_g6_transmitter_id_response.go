package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetG6TransmitterIDResponse is the pump's answer to a
// SetG6TransmitterIDRequest: Status 0 means success.
type SetG6TransmitterIDResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var setG6TransmitterIDResponseProps = messages.Props{OpCode: -79, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	MinAPI: "MOBI_API_V3_5", Request: "SetG6TransmitterIdRequest"}

func init() {
	messages.Register("SetG6TransmitterIdResponse", &SetG6TransmitterIDResponse{}, setG6TransmitterIDResponseProps,
		messages.Constructor{Params: "int status", Func: NewSetG6TransmitterIDResponse},
	)
}

// NewSetG6TransmitterIDResponse builds the response.
func NewSetG6TransmitterIDResponse(status int32) *SetG6TransmitterIDResponse {
	m := &SetG6TransmitterIDResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *SetG6TransmitterIDResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setG6TransmitterIDResponseProps, raw)
	if len(raw) != setG6TransmitterIDResponseProps.Size {
		return messages.InvalidLength("SetG6TransmitterIdResponse", len(raw), setG6TransmitterIDResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
