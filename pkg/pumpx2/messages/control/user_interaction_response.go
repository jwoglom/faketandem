package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// UserInteractionResponse is the pump's answer to a UserInteractionRequest:
// Status 0 is success.
type UserInteractionResponse struct {
	messages.Base
	Status int32 `pumpx2:"status"`
}

var userInteractionResponseProps = messages.Props{OpCode: -123, Size: 1, Signed: true, Type: messages.Response, Characteristic: messages.Control,
	Request: "UserInteractionRequest"}

func init() {
	messages.Register("UserInteractionResponse", &UserInteractionResponse{}, userInteractionResponseProps,
		messages.Constructor{Params: "int status", Func: NewUserInteractionResponse},
	)
}

// NewUserInteractionResponse builds the response.
func NewUserInteractionResponse(status int32) *UserInteractionResponse {
	m := &UserInteractionResponse{Status: status}
	m.SetCargo([]byte{byte(status)})
	return m
}

// Parse implements messages.Message.
func (m *UserInteractionResponse) Parse(raw []byte) error {
	raw = messages.RemoveSignedRequestHmacBytes(userInteractionResponseProps, raw)
	if len(raw) != userInteractionResponseProps.Size {
		return messages.InvalidLength("UserInteractionResponse", len(raw), userInteractionResponseProps.Size)
	}
	m.SetCargo(raw)
	m.Status = int32(int8(raw[0]))
	return nil
}
