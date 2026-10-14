package currentstatus

import "github.com/jwoglom/faketandem/pkg/pumpx2/messages"

// TempRateStatusRequest asks for the temp basal rate in force and its ID.
type TempRateStatusRequest struct {
	messages.Base
}

var tempRateStatusRequestProps = messages.Props{OpCode: 30, Type: messages.Request, Response: "TempRateStatusResponse"}

func init() {
	messages.Register("TempRateStatusRequest", &TempRateStatusRequest{}, tempRateStatusRequestProps)
}

// NewTempRateStatusRequest is the request, which has no payload.
func NewTempRateStatusRequest() *TempRateStatusRequest {
	m := &TempRateStatusRequest{}
	m.SetCargo([]byte{})
	return m
}

// Parse implements messages.Message.
func (m *TempRateStatusRequest) Parse(raw []byte) error {
	if len(raw) != tempRateStatusRequestProps.Size {
		return messages.InvalidLength("TempRateStatusRequest", len(raw), tempRateStatusRequestProps.Size)
	}
	m.SetCargo(raw)
	return nil
}
