package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetIDPSettingsResponse is the pump's answer to a SetIDPSettingsRequest:
// Status 0 means success. pumpX2 builds its second byte as 2, as captured,
// and does not read it.
type SetIDPSettingsResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var setIDPSettingsResponseProps = messages.Props{OpCode: -83, Size: 2, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "SetIDPSettingsRequest"}

func init() {
	messages.Register("SetIDPSettingsResponse", &SetIDPSettingsResponse{}, setIDPSettingsResponseProps,
		messages.Constructor{Params: "int status", Func: NewSetIDPSettingsResponse},
	)
}

// NewSetIDPSettingsResponse builds the response.
func NewSetIDPSettingsResponse(status int32) *SetIDPSettingsResponse {
	m := &SetIDPSettingsResponse{Status: status}
	m.SetCargo([]byte{byte(status), 2})
	return m
}

// Parse implements messages.Message.
func (m *SetIDPSettingsResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setIDPSettingsResponseProps, raw)
	if len(raw) != setIDPSettingsResponseProps.Size {
		return messages.InvalidLength("SetIDPSettingsResponse", len(raw), setIDPSettingsResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
