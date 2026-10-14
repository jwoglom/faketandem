package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SuspendPumpingResponse is the pump's answer to a SuspendPumpingRequest:
// Status 0 is success.
type SuspendPumpingResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var suspendPumpingResponseProps = messages.Props{OpCode: -99, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	MinAPI: "MOBI_API_V3_5", SupportedDevices: "MOBI_ONLY", Request: "SuspendPumpingRequest"}

func init() {
	messages.Register("SuspendPumpingResponse", &SuspendPumpingResponse{}, suspendPumpingResponseProps,
		messages.Constructor{Params: "int status", Func: NewSuspendPumpingResponse},
	)
}

// NewSuspendPumpingResponse builds the response.
func NewSuspendPumpingResponse(status int32) *SuspendPumpingResponse {
	m := &SuspendPumpingResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *SuspendPumpingResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(suspendPumpingResponseProps, raw)
	if len(raw) != suspendPumpingResponseProps.Size {
		return messages.InvalidLength("SuspendPumpingResponse", len(raw), suspendPumpingResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
