package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetMaxBasalLimitResponse is the pump's answer to a SetMaxBasalLimitRequest:
// Status 0 is success.
type SetMaxBasalLimitResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var setMaxBasalLimitResponseProps = messages.Props{OpCode: -119, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "SetMaxBasalLimitRequest"}

func init() {
	messages.Register("SetMaxBasalLimitResponse", &SetMaxBasalLimitResponse{}, setMaxBasalLimitResponseProps,
		messages.Constructor{Params: "int status", Func: NewSetMaxBasalLimitResponse},
	)
}

// NewSetMaxBasalLimitResponse builds the response.
func NewSetMaxBasalLimitResponse(status int32) *SetMaxBasalLimitResponse {
	m := &SetMaxBasalLimitResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *SetMaxBasalLimitResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setMaxBasalLimitResponseProps, raw)
	if len(raw) != setMaxBasalLimitResponseProps.Size {
		return messages.InvalidLength("SetMaxBasalLimitResponse", len(raw), setMaxBasalLimitResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
