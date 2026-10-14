package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// CurrentBasalStatusRequest asks for the basal rate in force and the profile's.
type CurrentBasalStatusRequest struct {
	messages.Base
}

var currentBasalStatusRequestProps = messages.Props{OpCode: 40, Type: messages.Request, Response: "CurrentBasalStatusResponse"}

func init() {
	messages.Register("CurrentBasalStatusRequest", &CurrentBasalStatusRequest{}, currentBasalStatusRequestProps,
		messages.Constructor{Func: NewCurrentBasalStatusRequest},
	)
}

// NewCurrentBasalStatusRequest is the request, which has no payload.
func NewCurrentBasalStatusRequest() *CurrentBasalStatusRequest {
	m := &CurrentBasalStatusRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *CurrentBasalStatusRequest) Parse(raw []byte) error {
	if len(raw) != currentBasalStatusRequestProps.Size {
		return messages.InvalidLength("CurrentBasalStatusRequest", len(raw), currentBasalStatusRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
