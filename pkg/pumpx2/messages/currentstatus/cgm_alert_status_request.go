package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CGMAlertStatusRequest asks which CGM alerts are active.
type CGMAlertStatusRequest struct {
	messages.Base
}

var cgmAlertStatusRequestProps = messages.Props{OpCode: 74, Type: messages.Request, Response: "CGMAlertStatusResponse"}

func init() {
	messages.Register("CGMAlertStatusRequest", &CGMAlertStatusRequest{}, cgmAlertStatusRequestProps,
		messages.Constructor{Func: NewCGMAlertStatusRequest},
	)
}

// NewCGMAlertStatusRequest is the request, which has no payload.
func NewCGMAlertStatusRequest() *CGMAlertStatusRequest {
	m := &CGMAlertStatusRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *CGMAlertStatusRequest) Parse(raw []byte) error {
	if len(raw) != cgmAlertStatusRequestProps.Size {
		return messages.InvalidLength("CGMAlertStatusRequest", len(raw), cgmAlertStatusRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
