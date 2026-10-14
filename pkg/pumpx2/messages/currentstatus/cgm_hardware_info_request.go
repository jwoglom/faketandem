package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CGMHardwareInfoRequest asks for the CGM transmitter's ID.
type CGMHardwareInfoRequest struct {
	messages.Base
}

var cgmHardwareInfoRequestProps = messages.Props{OpCode: 96, Type: messages.Request, Response: "CGMHardwareInfoResponse"}

func init() {
	messages.Register("CGMHardwareInfoRequest", &CGMHardwareInfoRequest{}, cgmHardwareInfoRequestProps,
		messages.Constructor{Func: NewCGMHardwareInfoRequest},
	)
}

// NewCGMHardwareInfoRequest is the request, which has no payload.
func NewCGMHardwareInfoRequest() *CGMHardwareInfoRequest {
	m := &CGMHardwareInfoRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. pumpX2 takes any payload.
func (m *CGMHardwareInfoRequest) Parse(raw []byte) error {
	m.SetCargo(raw)
	return nil
}
