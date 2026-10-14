package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetActiveIDPResponse is the pump's answer to a SetActiveIDPRequest: Status
// 0 means the profile is active.
type SetActiveIDPResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var setActiveIDPResponseProps = messages.Props{OpCode: -19, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, SupportedDevices: "MOBI_ONLY", Request: "SetActiveIDPRequest"}

func init() {
	messages.Register("SetActiveIDPResponse", &SetActiveIDPResponse{}, setActiveIDPResponseProps,
		messages.Constructor{Params: "int status", Func: NewSetActiveIDPResponse},
	)
}

// NewSetActiveIDPResponse builds the response.
func NewSetActiveIDPResponse(status int32) *SetActiveIDPResponse {
	m := &SetActiveIDPResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *SetActiveIDPResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setActiveIDPResponseProps, raw)
	if len(raw) != setActiveIDPResponseProps.Size {
		return messages.InvalidLength("SetActiveIDPResponse", len(raw), setActiveIDPResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
