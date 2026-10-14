package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// PumpVersionBRequest asks for the pump's software name and its serial,
// model and circuit board numbers.
type PumpVersionBRequest struct {
	messages.Base
}

var pumpVersionBRequestProps = messages.Props{OpCode: -124, Type: messages.Request, Response: "PumpVersionBResponse"}

func init() {
	messages.Register("PumpVersionBRequest", &PumpVersionBRequest{}, pumpVersionBRequestProps)
}

// NewPumpVersionBRequest is the request, which has no payload.
func NewPumpVersionBRequest() *PumpVersionBRequest {
	m := &PumpVersionBRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message. pumpX2 checks no length here.
func (m *PumpVersionBRequest) Parse(raw []byte) error {
	m.SetCargo(raw)
	return nil
}
