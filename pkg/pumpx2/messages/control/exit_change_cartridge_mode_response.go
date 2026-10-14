package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ExitChangeCartridgeModeResponse is the pump's answer to an
// ExitChangeCartridgeModeRequest: Status 0 is success.
type ExitChangeCartridgeModeResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var exitChangeCartridgeModeResponseProps = messages.Props{OpCode: -109, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control, Request: "ExitChangeCartridgeModeRequest"}

func init() {
	messages.Register("ExitChangeCartridgeModeResponse", &ExitChangeCartridgeModeResponse{}, exitChangeCartridgeModeResponseProps,
		messages.Constructor{Params: "int status", Func: NewExitChangeCartridgeModeResponse},
	)
}

// NewExitChangeCartridgeModeResponse builds the response.
func NewExitChangeCartridgeModeResponse(status int32) *ExitChangeCartridgeModeResponse {
	m := &ExitChangeCartridgeModeResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *ExitChangeCartridgeModeResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(exitChangeCartridgeModeResponseProps, raw)
	if len(raw) != exitChangeCartridgeModeResponseProps.Size {
		return messages.InvalidLength("ExitChangeCartridgeModeResponse", len(raw), exitChangeCartridgeModeResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
