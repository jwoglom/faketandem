package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// PumpVersionRequest asks for the pump's software versions and its serial,
// part and model numbers.
type PumpVersionRequest struct {
	messages.Base
}

var pumpVersionRequestProps = messages.Props{OpCode: 84, Type: messages.Request, Response: "PumpVersionResponse"}

func init() {
	messages.Register("PumpVersionRequest", &PumpVersionRequest{}, pumpVersionRequestProps)
}

// NewPumpVersionRequest is the request, which has no payload.
func NewPumpVersionRequest() *PumpVersionRequest {
	m := &PumpVersionRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *PumpVersionRequest) Parse(raw []byte) error {
	if len(raw) != pumpVersionRequestProps.Size {
		return messages.InvalidLength("PumpVersionRequest", len(raw), pumpVersionRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
