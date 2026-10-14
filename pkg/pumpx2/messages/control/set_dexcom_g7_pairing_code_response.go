package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SetDexcomG7PairingCodeResponse is the pump's answer to a
// SetDexcomG7PairingCodeRequest: Status 0 means success.
type SetDexcomG7PairingCodeResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var setDexcomG7PairingCodeResponseProps = messages.Props{OpCode: -3, Size: 2, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "SetDexcomG7PairingCodeRequest"}

func init() {
	messages.Register("SetDexcomG7PairingCodeResponse", &SetDexcomG7PairingCodeResponse{}, setDexcomG7PairingCodeResponseProps,
		messages.Constructor{Params: "int status", Func: NewSetDexcomG7PairingCodeResponse},
	)
}

// NewSetDexcomG7PairingCodeResponse builds the response.
func NewSetDexcomG7PairingCodeResponse(status int32) *SetDexcomG7PairingCodeResponse {
	m := &SetDexcomG7PairingCodeResponse{Status: status}
	m.SetCargo([]byte{byte(status), 0})
	return m
}

// Parse implements messages.Message.
func (m *SetDexcomG7PairingCodeResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(setDexcomG7PairingCodeResponseProps, raw)
	if len(raw) != setDexcomG7PairingCodeResponseProps.Size {
		return messages.InvalidLength("SetDexcomG7PairingCodeResponse", len(raw), setDexcomG7PairingCodeResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
