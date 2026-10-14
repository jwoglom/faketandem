package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// ExtendedBolusStatusRequest asks for the extended bolus in progress, if any.
type ExtendedBolusStatusRequest struct {
	messages.Base
}

var extendedBolusStatusRequestProps = messages.Props{OpCode: 46, Type: messages.Request, Response: "ExtendedBolusStatusResponse"}

func init() {
	messages.Register("ExtendedBolusStatusRequest", &ExtendedBolusStatusRequest{}, extendedBolusStatusRequestProps,
		messages.Constructor{Func: NewExtendedBolusStatusRequest},
	)
}

// NewExtendedBolusStatusRequest is the request, which has no payload.
func NewExtendedBolusStatusRequest() *ExtendedBolusStatusRequest {
	m := &ExtendedBolusStatusRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *ExtendedBolusStatusRequest) Parse(raw []byte) error {
	if len(raw) != extendedBolusStatusRequestProps.Size {
		return messages.InvalidLength("ExtendedBolusStatusRequest", len(raw), extendedBolusStatusRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
