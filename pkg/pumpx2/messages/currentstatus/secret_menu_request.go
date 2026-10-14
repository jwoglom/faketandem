package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// SecretMenuRequest asks for what the pump's secret menu shows, such as when
// it last connected.
type SecretMenuRequest struct {
	messages.Base
}

var secretMenuRequestProps = messages.Props{OpCode: -68, Type: messages.Request, Response: "SecretMenuResponse"}

func init() {
	messages.Register("SecretMenuRequest", &SecretMenuRequest{}, secretMenuRequestProps)
}

// NewSecretMenuRequest is the request, which has no payload.
func NewSecretMenuRequest() *SecretMenuRequest {
	m := &SecretMenuRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. pumpX2 checks no length here.
func (m *SecretMenuRequest) Parse(raw []byte) error {
	m.SetCargo(raw)
	return nil
}
