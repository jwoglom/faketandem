package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// PumpGlobalsRequest asks for the pump's quick bolus settings and how it
// announces each kind of event.
type PumpGlobalsRequest struct {
	messages.Base
}

var pumpGlobalsRequestProps = messages.Props{OpCode: 86, Type: messages.Request, Response: "PumpGlobalsResponse"}

func init() {
	messages.Register("PumpGlobalsRequest", &PumpGlobalsRequest{}, pumpGlobalsRequestProps)
}

// NewPumpGlobalsRequest is the request, which has no payload.
func NewPumpGlobalsRequest() *PumpGlobalsRequest {
	m := &PumpGlobalsRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *PumpGlobalsRequest) Parse(raw []byte) error {
	if len(raw) != pumpGlobalsRequestProps.Size {
		return messages.InvalidLength("PumpGlobalsRequest", len(raw), pumpGlobalsRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
