package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// LoadStatusRequest asks where the pump is in changing the cartridge and
// filling the tubing and cannula.
type LoadStatusRequest struct {
	messages.Base
}

var loadStatusRequestProps = messages.Props{OpCode: 20, Type: messages.Request, Response: "LoadStatusResponse"}

func init() {
	messages.Register("LoadStatusRequest", &LoadStatusRequest{}, loadStatusRequestProps,
		messages.Constructor{Func: NewLoadStatusRequest},
	)
}

// NewLoadStatusRequest is the request, which has no payload.
func NewLoadStatusRequest() *LoadStatusRequest {
	m := &LoadStatusRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *LoadStatusRequest) Parse(raw []byte) error {
	if len(raw) != loadStatusRequestProps.Size {
		return messages.InvalidLength("LoadStatusRequest", len(raw), loadStatusRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
