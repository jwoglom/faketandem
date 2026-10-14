package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CGMStatusRequest asks for the CGM session's state and the transmitter
// battery.
type CGMStatusRequest struct {
	messages.Base
}

var cgmStatusRequestProps = messages.Props{OpCode: 80, Type: messages.Request, Response: "CGMStatusResponse"}

func init() {
	messages.Register("CGMStatusRequest", &CGMStatusRequest{}, cgmStatusRequestProps,
		messages.Constructor{Func: NewCGMStatusRequest},
	)
}

// NewCGMStatusRequest is the request, which has no payload.
func NewCGMStatusRequest() *CGMStatusRequest {
	m := &CGMStatusRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *CGMStatusRequest) Parse(raw []byte) error {
	if len(raw) != cgmStatusRequestProps.Size {
		return messages.InvalidLength("CGMStatusRequest", len(raw), cgmStatusRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
