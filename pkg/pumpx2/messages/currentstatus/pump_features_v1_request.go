package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// PumpFeaturesV1Request asks which features the pump supports.
type PumpFeaturesV1Request struct {
	messages.Base
}

var pumpFeaturesV1RequestProps = messages.Props{OpCode: 78, Type: messages.Request, Response: "PumpFeaturesV1Response"}

func init() {
	messages.Register("PumpFeaturesV1Request", &PumpFeaturesV1Request{}, pumpFeaturesV1RequestProps)
}

// NewPumpFeaturesV1Request is the request, which has no payload.
func NewPumpFeaturesV1Request() *PumpFeaturesV1Request {
	m := &PumpFeaturesV1Request{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *PumpFeaturesV1Request) Parse(raw []byte) error {
	if len(raw) != pumpFeaturesV1RequestProps.Size {
		return messages.InvalidLength("PumpFeaturesV1Request", len(raw), pumpFeaturesV1RequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
