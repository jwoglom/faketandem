package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// BleSoftwareInfoRequest asks for the version of the pump's Bluetooth stack.
type BleSoftwareInfoRequest struct {
	messages.Base
}

var bleSoftwareInfoRequestProps = messages.Props{OpCode: -120, Type: messages.Request, Response: "BleSoftwareInfoResponse"}

func init() {
	messages.Register("BleSoftwareInfoRequest", &BleSoftwareInfoRequest{}, bleSoftwareInfoRequestProps,
		messages.Constructor{Func: NewBleSoftwareInfoRequest},
	)
}

// NewBleSoftwareInfoRequest is the request, which has no payload.
func NewBleSoftwareInfoRequest() *BleSoftwareInfoRequest {
	m := &BleSoftwareInfoRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. pumpX2 takes any payload.
func (m *BleSoftwareInfoRequest) Parse(raw []byte) error {
	m.SetCargo(raw)
	return nil
}
