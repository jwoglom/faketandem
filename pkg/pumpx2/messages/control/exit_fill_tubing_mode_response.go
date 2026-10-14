package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ExitFillTubingModeResponse is the pump's answer to an
// ExitFillTubingModeRequest: Status 0 is success. The pump has left fill tubing
// mode only once an ExitFillTubingModeStateStreamResponse says so.
type ExitFillTubingModeResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var exitFillTubingModeResponseProps = messages.Props{OpCode: -105, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control, Request: "ExitFillTubingModeRequest"}

func init() {
	messages.Register("ExitFillTubingModeResponse", &ExitFillTubingModeResponse{}, exitFillTubingModeResponseProps,
		messages.Constructor{Params: "int status", Func: NewExitFillTubingModeResponse},
	)
}

// NewExitFillTubingModeResponse builds the response.
func NewExitFillTubingModeResponse(status int32) *ExitFillTubingModeResponse {
	m := &ExitFillTubingModeResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *ExitFillTubingModeResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(exitFillTubingModeResponseProps, raw)
	if len(raw) != exitFillTubingModeResponseProps.Size {
		return messages.InvalidLength("ExitFillTubingModeResponse", len(raw), exitFillTubingModeResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
