package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetMaxBolusLimitResponse is the pump's answer to a SetMaxBolusLimitRequest:
// Status 0 is success.
type SetMaxBolusLimitResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var setMaxBolusLimitResponseProps = messages.Props{OpCode: -121, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "SetMaxBolusLimitRequest"}

func init() {
	messages.Register("SetMaxBolusLimitResponse", &SetMaxBolusLimitResponse{}, setMaxBolusLimitResponseProps,
		messages.Constructor{Params: "int status", Func: NewSetMaxBolusLimitResponse},
	)
}

// NewSetMaxBolusLimitResponse builds the response.
func NewSetMaxBolusLimitResponse(status int32) *SetMaxBolusLimitResponse {
	m := &SetMaxBolusLimitResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *SetMaxBolusLimitResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setMaxBolusLimitResponseProps, raw)
	if len(raw) != setMaxBolusLimitResponseProps.Size {
		return messages.InvalidLength("SetMaxBolusLimitResponse", len(raw), setMaxBolusLimitResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
