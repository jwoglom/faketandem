package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// EnterChangeCartridgeModeResponse is the pump's answer to an
// EnterChangeCartridgeModeRequest: Status 0 means it entered change cartridge
// mode.
type EnterChangeCartridgeModeResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var enterChangeCartridgeModeResponseProps = messages.Props{OpCode: -111, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	ModifiesInsulinDelivery: true, Request: "EnterChangeCartridgeModeRequest"}

func init() {
	messages.Register("EnterChangeCartridgeModeResponse", &EnterChangeCartridgeModeResponse{}, enterChangeCartridgeModeResponseProps,
		messages.Constructor{Params: "int status", Func: NewEnterChangeCartridgeModeResponse},
		messages.Constructor{Params: "byte[] raw", Func: NewEnterChangeCartridgeModeResponseFromCargo},
	)
}

// NewEnterChangeCartridgeModeResponse builds the response.
func NewEnterChangeCartridgeModeResponse(status int32) *EnterChangeCartridgeModeResponse {
	m := &EnterChangeCartridgeModeResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// NewEnterChangeCartridgeModeResponseFromCargo parses a response's payload.
func NewEnterChangeCartridgeModeResponseFromCargo(raw []byte) (*EnterChangeCartridgeModeResponse, error) {
	m := &EnterChangeCartridgeModeResponse{}
	return m, m.Parse(raw)
}

// Parse implements messages.Message.
func (m *EnterChangeCartridgeModeResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(enterChangeCartridgeModeResponseProps, raw)
	if len(raw) != enterChangeCartridgeModeResponseProps.Size {
		return messages.InvalidLength("EnterChangeCartridgeModeResponse", len(raw), enterChangeCartridgeModeResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
