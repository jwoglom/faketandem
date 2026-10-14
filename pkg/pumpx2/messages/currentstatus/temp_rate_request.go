package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// TempRateRequest asks for the temp basal rate in force.
type TempRateRequest struct {
	messages.Base
}

var tempRateRequestProps = messages.Props{OpCode: 42, Type: messages.Request, Response: "TempRateResponse"}

func init() {
	messages.Register("TempRateRequest", &TempRateRequest{}, tempRateRequestProps)
}

// NewTempRateRequest is the request, which has no payload.
func NewTempRateRequest() *TempRateRequest {
	m := &TempRateRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *TempRateRequest) Parse(raw []byte) error {
	if len(raw) != tempRateRequestProps.Size {
		return messages.InvalidLength("TempRateRequest", len(raw), tempRateRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
