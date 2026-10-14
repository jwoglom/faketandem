package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CurrentBolusStatusRequest asks for the bolus being requested or delivered, if
// any.
type CurrentBolusStatusRequest struct {
	messages.Base
}

var currentBolusStatusRequestProps = messages.Props{OpCode: 44, Type: messages.Request, Response: "CurrentBolusStatusResponse"}

func init() {
	messages.Register("CurrentBolusStatusRequest", &CurrentBolusStatusRequest{}, currentBolusStatusRequestProps,
		messages.Constructor{Func: NewCurrentBolusStatusRequest},
	)
}

// NewCurrentBolusStatusRequest is the request, which has no payload.
func NewCurrentBolusStatusRequest() *CurrentBolusStatusRequest {
	m := &CurrentBolusStatusRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *CurrentBolusStatusRequest) Parse(raw []byte) error {
	if len(raw) != currentBolusStatusRequestProps.Size {
		return messages.InvalidLength("CurrentBolusStatusRequest", len(raw), currentBolusStatusRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
