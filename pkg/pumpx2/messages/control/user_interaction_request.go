package control

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// UserInteractionRequest is an empty command whose effect on the pump
// pumpX2 does not document.
type UserInteractionRequest struct {
	messages.Base
}

var userInteractionRequestProps = messages.Props{OpCode: -124, Signed: true, Type: messages.Request, Characteristic: messages.Control,
	Response: "UserInteractionResponse"}

func init() {
	messages.Register("UserInteractionRequest", &UserInteractionRequest{}, userInteractionRequestProps,
		messages.Constructor{Func: NewUserInteractionRequest},
	)
}

// NewUserInteractionRequest is the request, with no payload.
func NewUserInteractionRequest() *UserInteractionRequest {
	m := &UserInteractionRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. Unlike the other empty requests, it
// takes a payload of any length.
func (m *UserInteractionRequest) Parse(raw []byte) error {
	m.SetCargo(messages.RemoveSignedRequestHmacBytes(userInteractionRequestProps, raw))
	return nil
}
