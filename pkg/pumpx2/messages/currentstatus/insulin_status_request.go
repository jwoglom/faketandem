package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// InsulinStatusRequest asks how much insulin is left in the cartridge.
type InsulinStatusRequest struct {
	messages.Base
}

var insulinStatusRequestProps = messages.Props{OpCode: 36, Type: messages.Request, Response: "InsulinStatusResponse"}

func init() {
	messages.Register("InsulinStatusRequest", &InsulinStatusRequest{}, insulinStatusRequestProps,
		messages.Constructor{Func: NewInsulinStatusRequest},
	)
}

// NewInsulinStatusRequest is the request, which has no payload.
func NewInsulinStatusRequest() *InsulinStatusRequest {
	m := &InsulinStatusRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *InsulinStatusRequest) Parse(raw []byte) error {
	if len(raw) != insulinStatusRequestProps.Size {
		return messages.InvalidLength("InsulinStatusRequest", len(raw), insulinStatusRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
